package discordkvs

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/bwmarrin/discordgo"
	"golang.org/x/crypto/pbkdf2"
	"io/ioutil"
)

type ApplicationConfigOption int

const (
	AcceptDataFromOtherUsers ApplicationConfigOption = iota
)

var ErrNoExist = errors.New("pair does not exist")

const KVSChannelName = "discordkvs_donotremove"

type Application struct {
	s                        *discordgo.Session
	id                       []byte
	acceptDataFromOtherUsers bool
	key                      []byte
	block                    cipher.Block
	aesGCM                   cipher.AEAD
}

func NewApplication(s *discordgo.Session, id string, opts ...ApplicationConfigOption) (*Application, error) {
	salt := make([]byte, 16)

	_, err := rand.Read(salt)

	if err != nil {
		return nil, err
	}

	salt = salt[:16]

	key := pbkdf2.Key([]byte(id), salt, 13000, 32, sha256.New)

	block, err := aes.NewCipher(key)

	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)

	if err != nil {
		return nil, err
	}

	a := &Application{
		s:                        s,
		id:                       []byte(id),
		acceptDataFromOtherUsers: false,
		key:                      key,
		block:                    block,
		aesGCM:                   gcm,
	}

	for _, o := range opts {
		switch o {
		case AcceptDataFromOtherUsers:
			a.acceptDataFromOtherUsers = true
		default:
			panic(errors.New("unknown ApplicationConfigOption"))
		}
	}

	return a, nil
}

// Gets the KVS channel for guild, creating one if it doesn't already exist.
func (a *Application) GetKVSChannel(guildID string) (*discordgo.Channel, error) {
	guildChannels, err := a.s.GuildChannels(guildID)

	if err != nil {
		return nil, err
	}

	var kvsChannel *discordgo.Channel

	for _, c := range guildChannels {
		if c.Name == KVSChannelName {
			kvsChannel = c
		}
	}

	if kvsChannel != nil {
		// Already exist.

		return kvsChannel, nil
	}

	guild, err := a.s.Guild(guildID)

	if err != nil {
		return nil, err
	}

	createdChannel, err := a.s.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name:     KVSChannelName,
		Type:     discordgo.ChannelTypeGuildText,
		Topic:    "This is an automated channel. Please allow bots to access it. Feel free to mute it.",
		Position: 0,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			&discordgo.PermissionOverwrite{
				ID:    a.s.State.User.ID,
				Type:  "1",
				Deny:  0,
				Allow: discordgo.PermissionAllText,
			},
		},
		NSFW: false,
	})

	if err != nil {
		return nil, err
	}

	a.s.ChannelMessageSend(createdChannel.ID, "Hello there, <@"+guild.OwnerID+">. This is an automated channel for bots to use for storing data.\n**Please ensure that bots are able to access this channel!**\nIf the channel is deleted, you may lose bot data such as configurations.\n\nTo avoid notifications from this channel, you may mute it and hide it from non-bot members.\nFor more information, see the repository: https://github.com/ethanent/discordkvs")

	return createdChannel, nil
}

func (a *Application) keyHashStr(key string) string {
	hashed := sha256.Sum256(append([]byte(key), []byte(a.id)...))

	return hex.EncodeToString(hashed[:])
}

func (a *Application) Set(guildID string, key string, value []byte) error {
	kvs, err := a.GetKVSChannel(guildID)

	if err != nil {
		return err
	}

	hashed := a.keyHashStr(key)

	nonce := make([]byte, a.aesGCM.NonceSize())

	_, err = rand.Read(nonce)

	if err != nil {
		return err
	}

	enc := a.aesGCM.Seal(nil, nonce, value, []byte{})

	// Decide on dataLoc

	var dataLoc = dataInAttachment

	if len(enc)*2+len(hashed)+3 < 1999 {
		dataLoc = dataInContent
	}

	msgContent := serializeMessageContent(&messageContentMetadata{
		KeyHashStr: hashed,
		DataLoc:    dataLoc,
		Nonce:      nonce,
		Data:       enc,
	})

	// Get sending, method depends on dataLoc.

	var addedMessage *discordgo.Message

	if dataLoc == dataInAttachment {
		r := bytes.NewReader(enc)

		addedMessage, err = a.s.ChannelMessageSendComplex(kvs.ID, &discordgo.MessageSend{
			Content: msgContent,
			File: &discordgo.File{
				Name:        hex.EncodeToString(nonce),
				ContentType: "application/octet-stream",
				Reader:      r,
			},
		})

		if err != nil {
			return err
		}
	} else if dataLoc == dataInContent {
		addedMessage, err = a.s.ChannelMessageSendComplex(kvs.ID, &discordgo.MessageSend{
			Content: msgContent,
		})
	}

	// Clear old values, if 1/10 chance met
	// The chance is for performance. Cleaning up is slow and probably affects ratelimiting.

	if nonce[0] < 255/10 {
		old, err := a.getMessages(kvs, -1, 100, &filterDescriptor{
			by:       filterByKeyHash,
			selector: hashed,
		})

		// Let's clear if no error. If error, let's ignore it. Not the end of the world.
		if err == nil {
			// fmt.Println(old)

			mids := make([]string, 0, len(old))

			for _, m := range old {
				if m.ID == addedMessage.ID {
					continue
				}

				mids = append(mids, m.ID)
			}

			if err := a.s.ChannelMessagesBulkDelete(kvs.ID, mids); err != nil {
				// Ignore error, but hopefully this doesn't happen.
				// Channel will get crowded if bot can't delete old values.
			}
		}
	}

	return nil
}

func (a *Application) Get(guildID string, key string) ([]byte, error) {
	kvsChannel, err := a.GetKVSChannel(guildID)

	if err != nil {
		return nil, err
	}

	keyHash := a.keyHashStr(key)

	res, err := a.getMessages(kvsChannel, 1, -1, &filterDescriptor{
		by:       filterByKeyHash,
		selector: keyHash,
	})

	if err != nil {
		return nil, err
	}

	// Error if not exist

	if len(res) == 0 {
		return nil, ErrNoExist
	}

	// Unpack data, first parse content

	dataMsg := res[0]

	parsed, err := parseMessageContent(dataMsg.Content)

	if err != nil {
		// This shouldn't happen as the getMessages method should run this too

		return nil, errors.New("invalid message content, cannot find dataLoc")
	}

	var d []byte

	if parsed.DataLoc == dataInAttachment {
		// Error if invalid

		if len(dataMsg.Attachments) < 1 {
			return nil, errors.New("invalid data message")
		}

		// Continue unpacking data

		attach := dataMsg.Attachments[0]

		// Fetch data body

		resp, err := a.s.Client.Get(attach.URL)

		if err != nil {
			return nil, err
		}

		if resp.StatusCode != 200 {
			return nil, errors.New("unexpected status code")
		}

		defer resp.Body.Close()

		d, err = ioutil.ReadAll(resp.Body)

		if err != nil {
			return nil, err
		}
	} else if parsed.DataLoc == dataInContent {
		d = parsed.Data
	}

	// Decrypt body

	decrypted, err := a.aesGCM.Open(nil, parsed.Nonce, d, []byte{})

	if err != nil {
		return nil, err
	}

	return decrypted, nil
}

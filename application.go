package discordkvs

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/bwmarrin/discordgo"
	"github.com/ethanent/randbytes"
	"golang.org/x/crypto/pbkdf2"
)

type ApplicationConfigOption int

const (
	AcceptDataFromOtherUsers ApplicationConfigOption = iota
)

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
	salt, err := randbytes.RandBytes(16)

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

	nonce, err := randbytes.RandBytes(a.aesGCM.NonceSize())

	if err != nil {
		return err
	}

	enc := a.aesGCM.Seal(nil, nonce, value, []byte{})

	r := bytes.NewReader(enc)

	_, err = a.s.ChannelMessageSendComplex(kvs.ID, &discordgo.MessageSend{
		Content: hashed,
		File: &discordgo.File{
			Name:        "d",
			ContentType: "application/octet-stream",
			Reader:      r,
		},
	})

	if err != nil {
		return err
	}

	return nil
}

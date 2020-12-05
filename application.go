package discordkvs

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/bwmarrin/discordgo"
	"golang.org/x/crypto/pbkdf2"
)

type ApplicationConfigOption func(*Application)

func AcceptDataFromOtherUsers(a *Application) {
	a.acceptDataFromOtherUsers = true
}

var ErrNoExist = errors.New("pair does not exist")

const KVSChannelName = "discordkvs_donotremove"

var kvsChannelIDCache = map[string]string{}

type Application struct {
	s                        *discordgo.Session
	id                       []byte
	acceptDataFromOtherUsers bool
	key                      []byte
	block                    cipher.Block
	aesGCM                   cipher.AEAD
}

func NewApplication(s *discordgo.Session, id string, opts ...ApplicationConfigOption) (*Application, error) {
	// The same salt is used each time because the key output should be
	// deterministic. (This is so that the consumer can treat the Application ID like
	// a password if they wish.
	key := pbkdf2.Key([]byte(id), []byte{52, 62, 10, 123, 240, 138, 71, 183}, 13000, 32, sha256.New)

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
		o(a)
	}

	return a, nil
}

// Gets the KVS channel for guild, creating one if it doesn't already exist.
func (a *Application) GetKVSChannel(guildID string) (*discordgo.Channel, error) {
	// First, check cache

	cachedID, ok := kvsChannelIDCache[guildID]

	if ok {
		fetchedChannel, err := a.s.Channel(cachedID)

		if err != nil {
			// Purge channel from cache and continue. Channel fetch errored.

			delete(kvsChannelIDCache, guildID)
		} else {
			return fetchedChannel, nil
		}
	}

	// Not cached, locate channel.

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

		// Ensure cached.

		kvsChannelIDCache[guildID] = kvsChannel.ID

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

	// Cache created KVS channel.

	kvsChannelIDCache[guildID] = createdChannel.ID

	return createdChannel, nil
}

func (a *Application) keyHashStr(key string) string {
	hashed := sha256.Sum256(append([]byte(key), []byte(a.id)...))

	return hex.EncodeToString(hashed[:])
}

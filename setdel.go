package discordkvs

import (
	"bytes"
	"crypto/rand"
	"errors"
	"github.com/bwmarrin/discordgo"
)

var ErrEmptyKey = errors.New("empty keys are not permitted")

// Set writes a key-value pair.
func (a *Application) Set(guildID string, key string, value []byte) error {
	if len(key) == 0 {
		return ErrEmptyKey
	}

	kvsChannelID, err := a.GetKVSChannelID(guildID)

	if err != nil {
		return err
	}

	hashedKey := a.keyHashStr(key)

	nonce := make([]byte, a.aesGCM.NonceSize())

	_, err = rand.Read(nonce)

	if err != nil {
		return err
	}

	enc := a.aesGCM.Seal(nil, nonce, value, []byte{})

	// Decide on dataLoc

	var dataLoc = dataInAttachment

	if len(enc)*2+len(hashedKey)+3 < 1999 {
		dataLoc = dataInContent
	}

	msgContent := serializeMessageContent(&messageContentMetadata{
		KeyHashStr: hashedKey,
		DataLoc:    dataLoc,
		Nonce:      nonce,
		Data:       enc,
	})

	// Send pair data, method depends on dataLoc.

	var addedMessage *discordgo.Message

	if dataLoc == dataInAttachment {
		r := bytes.NewReader(enc)

		addedMessage, err = a.s.ChannelMessageSendComplex(kvsChannelID, &discordgo.MessageSend{
			Content: msgContent,
			File: &discordgo.File{
				Name:        "d",
				ContentType: "application/octet-stream",
				Reader:      r,
			},
		})
	} else if dataLoc == dataInContent {
		addedMessage, err = a.s.ChannelMessageSendComplex(kvsChannelID, &discordgo.MessageSend{
			Content: msgContent,
		})
	}

	if err != nil {
		// Eliminate KVS channel ID from cache, in case it doesn't exist anymore.
		delete(kvsChannelIDCache, guildID)

		return err
	}

	// Clear old values, if 1/10 chance met
	// The chance is for performance. Cleaning up is slow and probably affects ratelimiting.

	if nonce[0] < 255/10 {
		old, err := a.getMessages(kvsChannelID, -1, 100, &filterDescriptor{
			by:       filterByKeyHash,
			selector: hashedKey,
		})

		// Let's clear if no error. If error, let's ignore it. Not the end of the world.
		if err == nil {
			mids := make([]*discordgo.Message, 0, len(old))

			for _, m := range old {
				if m.ID == addedMessage.ID {
					continue
				}

				mids = append(mids, m)
			}

			if err := bulkDelete(a.s, kvsChannelID, mids); err != nil {
				// Ignore error, but hopefully this doesn't happen.
				// Channel will get crowded if bot can't delete old values.
			}
		}
	}

	return nil
}

// Del deletes a key-value pair.
func (a *Application) Del(guildID string, key string) error {
	kvsChannelID, err := a.GetKVSChannelID(guildID)

	if err != nil {
		return err
	}

	res, err := a.getMessages(kvsChannelID, -1, -1, &filterDescriptor{
		by:       filterByKeyHash,
		selector: a.keyHashStr(key),
	})

	if err != nil {
		// Eliminate KVS channel ID from cache, in case it doesn't exist anymore.
		delete(kvsChannelIDCache, guildID)

		return err
	}

	err = bulkDelete(a.s, kvsChannelID, res)

	return err
}

func bulkDelete(s *discordgo.Session, channelID string, messages []*discordgo.Message) error {
	var curGroup []string

	for _, m := range messages {
		curGroup = append(curGroup, m.ID)

		if len(curGroup) >= 100 {
			if err := s.ChannelMessagesBulkDelete(channelID, curGroup); err != nil {
				return err
			}

			curGroup = []string{}
		}
	}

	if len(curGroup) > 0 {
		if err := s.ChannelMessagesBulkDelete(channelID, curGroup); err != nil {
			return err
		}
	}

	return nil
}

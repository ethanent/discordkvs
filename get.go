package discordkvs

import (
	"errors"
	"io/ioutil"
)

// Get gets the corresponding value given a key.
func (a *Application) Get(guildID string, key string) ([]byte, error) {
	kvsChannelID, err := a.GetKVSChannelID(guildID)

	if err != nil {
		return nil, err
	}

	keyHash := a.keyHashStr(key)

	requireAuthorID := ""

	if !a.acceptDataFromOtherUsers {
		requireAuthorID = a.s.State.User.ID
	}

	res, err := a.getMessages(kvsChannelID, 1, -1, &filterDescriptor{
		by:              filterByKeyHash,
		selector:        keyHash,
		requireAuthorID: requireAuthorID,
	})

	if err != nil {
		// Eliminate KVS channel ID from cache, in case it doesn't exist anymore.
		delete(kvsChannelIDCache, guildID)

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

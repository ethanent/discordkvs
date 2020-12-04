package discordkvs

import (
	"encoding/hex"
	"errors"
	"strings"
)

type dataLocation int

const (
	dataInContent dataLocation = iota
	dataInAttachment
)

type messageContentMetadata struct {
	KeyHashStr string
	Nonce      []byte
	DataLoc    dataLocation
	Data       []byte
}

func parseMessageContent(c string) (*messageContentMetadata, error) {
	split := strings.Split(c, ",")

	if len(split) < 3 {
		return nil, errors.New("invalid message content, bad len")
	}

	r := &messageContentMetadata{
		KeyHashStr: split[0],
		DataLoc:    0,
	}

	// Parse nonce

	nonce, err := hex.DecodeString(split[2])

	if err != nil {
		return nil, err
	}

	r.Nonce = nonce

	// Parse DataLoc

	switch split[1] {
	case "0":
		r.DataLoc = dataInContent
	case "1":
		r.DataLoc = dataInAttachment
	default:
		return nil, errors.New("invalid message DataLoc specifier")
	}

	// Parse data if present

	if r.DataLoc == dataInContent && len(split) >= 4 {
		r.Data, err = hex.DecodeString(split[3])

		if err != nil {
			return nil, err
		}
	}

	return r, nil
}

func serializeMessageContent(d *messageContentMetadata) string {
	ser := d.KeyHashStr + ","

	switch d.DataLoc {
	case dataInContent:
		ser += "0"
	case dataInAttachment:
		ser += "1"
	default:
		panic("unknown DataLoc")
	}

	ser += "," + hex.EncodeToString(d.Nonce) + ","

	if d.DataLoc == dataInAttachment {
		ser += "n"
	} else {
		ser += hex.EncodeToString(d.Data)
	}

	return ser
}

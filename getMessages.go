package discordkvs

import (
	"github.com/bwmarrin/discordgo"
)

type filterMessagesMethod int

const (
	filterByKeyHash filterMessagesMethod = iota
	filterByUserID
)

type filterDescriptor struct {
	by              filterMessagesMethod
	selector        string
	requireAuthorID string
}

func (a *Application) getMessages(kvsChannelID string, limit int, maxSearch int, method *filterDescriptor) ([]*discordgo.Message, error) {
	useCap := limit

	if useCap == -1 {
		useCap = 0
	}

	discoveredRelevant := make([]*discordgo.Message, 0, useCap)

	var earliestMsg *discordgo.Message
	searched := 0

	for len(discoveredRelevant) < limit || limit == -1 {
		var msgs []*discordgo.Message
		var err error

		if earliestMsg != nil {
			msgs, err = a.s.ChannelMessages(kvsChannelID, 100, earliestMsg.ID, "", "")
		} else {
			msgs, err = a.s.ChannelMessages(kvsChannelID, 100, "", "", "")
		}

		if err != nil {
			return nil, err
		}

		if len(msgs) == 0 {
			// Done with channel

			break
		}

		searched += len(msgs)

		earliestMsg = msgs[len(msgs)-1]

		for _, msg := range msgs {
			if method.by == filterByKeyHash {
				if method.requireAuthorID != "" {
					if msg.Author.ID != method.requireAuthorID {
						continue
					}
				}

				parsed, err := parseMessageContent(msg.Content)

				if err != nil {
					// Skip invalid message.

					continue
				}

				if parsed.KeyHashStr == method.selector {
					discoveredRelevant = append(discoveredRelevant, msg)
				}
			} else if method.by == filterByUserID {
				if msg.Author.ID == method.selector {
					discoveredRelevant = append(discoveredRelevant, msg)
				}
			} else {
				panic("unknown filter method")
			}
		}

		if maxSearch != -1 && searched >= maxSearch {
			break
		}
	}

	if limit != -1 && len(discoveredRelevant) > limit {
		return discoveredRelevant[:limit], nil
	} else {
		return discoveredRelevant, nil
	}
}

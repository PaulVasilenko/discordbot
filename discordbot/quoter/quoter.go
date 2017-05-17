package quoter

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"strconv"
)

type Quoter struct{}

func NewQuoter() *Quoter {
	return &Quoter{}
}

func (q *Quoter) Subscribe(s *discordgo.Session) {
	s.AddHandler(q.MessageReactionAdd)
}

func (q *Quoter) MessageReactionAdd(s *discordgo.Session, mr *discordgo.MessageReactionAdd) {
	if mr.Emoji.Name != "©" {
		return
	}

	convertedID, err := strconv.ParseInt(mr.MessageID, 10, 64)

	if err != nil {
		log.Println("Error converting ID: ", err)
		return
	}

	messageID := fmt.Sprintf("%v", convertedID-1)

	messages, err := s.ChannelMessages(mr.ChannelID, 100, "", messageID)

	if err != nil {
		log.Println("Error getting message: ", err)
		return
	}

	quoter, err := s.User(mr.UserID)

	if err != nil {
		log.Println("Error getting user: ", err)
		return
	}

	content := ""

	lastIndex := len(messages) - 1

	for i := lastIndex; i >= 0; i-- {
		if messages[i].Author.ID !=
			messages[lastIndex].Author.ID {
			break
		}

		content += messages[i].Content + "\n"
	}

	embed := &discordgo.MessageEmbed{
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Quoter", Value: quoter.Username},
			{Name: "Sender", Value: "<@" + messages[lastIndex].Author.ID + ">"},
			{Name: "Message", Value: content},
		},
	}

	_, err = s.ChannelMessageSendEmbed(mr.ChannelID, embed)

	if err != nil {
		log.Println("Error getting user: ", err)
		return
	}
}

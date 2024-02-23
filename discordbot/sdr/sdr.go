package sdr

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// SDR represents SDR plugin
type SDR struct {
	texts map[int]string
	gifts map[int]Gift
}

type User struct {
	UserID string     `bson:"user_id"`
	Gifts  []UserGift `bson:"gifts"`
}

type UserGift struct {
	Text string `bson:"text"`
	Gift Gift   `bson:"gift"`
}

type Gift struct {
	Image       string `bson:"image" json:"image"`
	Description string `bson:"description" json:"desc"`
}

func NewSDR(texts map[int]string, gifts map[int]Gift) (*SDR, error) {
	return &SDR{
		texts: texts,
		gifts: gifts,
	}, nil
}

func init() {
	rand.Seed(time.Now().Unix())
}

func (r *SDR) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !strings.HasPrefix(m.Message.Content, "!sdr") {
		return
	}

	if len(m.Mentions) == 0 {
		s.ChannelMessageSend(m.ChannelID, "mention user to give a gift")
		return
	}

	mentionedUser := m.Mentions[0]

	text := r.texts[rand.Intn(len(r.texts))]
	gift := r.gifts[rand.Intn(len(r.gifts))]

	// TODO: mongodb

	s.ChannelMessageSend(
		m.ChannelID,
		fmt.Sprintf(
			`%v присылает подарок для %v:
`+"```"+`
%v
`+"```"+`
И подарок:
%v
%v
`,
			m.Author.Mention(), mentionedUser.Mention(), text, gift.Image, gift.Description,
		),
	)
}

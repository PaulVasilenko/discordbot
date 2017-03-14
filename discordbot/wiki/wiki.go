package wiki

import (
	"github.com/bwmarrin/discordgo"
	"strings"
)

type Wiki struct{}

func NewWiki() *Wiki {
	return &Wiki{}
}

func (w *Wiki) Subscribe(s *discordgo.Session) {
	s.AddHandler(w.MessageCreate)
}

func (w *Wiki) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if strings.HasPrefix(m.Content, "!wiki") {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Processing wiki")
	}
}

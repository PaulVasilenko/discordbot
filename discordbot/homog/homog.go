package homog

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strings"
)

type Homog struct{}

func NewHomog() *Homog {
	return &Homog{}
}

func (h *Homog) Subscribe(s *discordgo.Session) {
	s.AddHandler(h.MessageCreate)
}

func (h *Homog) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	data := [3]string{"", "Гомогенезация", "женщины"}
	args := strings.Split(m.Content, " ")

	if args[0] != "!homog" {
		return
	}

	for index, value := range args {
		if index > 2 {
			data[2] += " " + value
		} else {
			data[index] = value
		}
	}

	_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s? Нет, спасибо, мне нравятся %s", data[1], data[2]))
}

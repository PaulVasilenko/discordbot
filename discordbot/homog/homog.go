// Package homog provides homogenezation to your discord chat
package homog

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strings"
)

// Homog is a struct representing plugin with it's main configurations
type Homog struct{}

// NewHomog is a constructor function for homog
func NewHomog() *Homog {
	return &Homog{}
}

// Subscribe is a method to subscribe to any events required by plugin
func (h *Homog) Subscribe(s *discordgo.Session) {
	s.AddHandler(h.MessageCreate)
}

// MessageCreate is a event method for message sent to discord
func (h *Homog) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	data := [2]string{"Гомогенезация", "женщины"}

	pattern := "%s? Нет, спасибо, мне нравятся %s"

	var message string

	switch {
	case strings.HasPrefix(m.Content, "!homog2"):
		pattern = "%s? Нет, спасибо, мне нравится %s"
		message = strings.Replace(m.Content, "!homog2", "", 1)
	case strings.HasPrefix(m.Content, "!homog"):
		message = strings.Replace(m.Content, "!homog", "", 1)
	default:
		return
	}

	args := strings.Split(message, "%")

	for index, value := range args {
		if value == "" {
			continue
		}
		data[index] = value
	}

	_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf(pattern, data[0], data[1]))
}

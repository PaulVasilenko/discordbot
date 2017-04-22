// Package shoutlikeart calculates whether you shout like art or not
package shoutlikeart

import (
	"github.com/bwmarrin/discordgo"
	"fmt"
	"time"
)

type ShoutLikeArt struct {}

func NewShoutLikeArt() *ShoutLikeArt {
	return &ShoutLikeArt{}
}

func (sla *ShoutLikeArt) Subscribe(dg *discordgo.Session) {
	dg.AddHandler(sla.MessageCreate)
}

func (sla *ShoutLikeArt) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Content != "!sla" {
		fmt.Println("WTF? ", m.Content)

		return
	}

	channel, err := s.Channel(m.ChannelID)

	if err != nil {
		fmt.Println("Unable to get channel info: ", err)

		return
	}

	guild, err := s.Guild(channel.GuildID)

	if err != nil {
		fmt.Println("Unable to get guild info: ", err)

		return
	}

	channelID := ""

	for _, vs := range guild.VoiceStates {
		if vs.UserID == m.Author.ID {
			channelID = vs.ChannelID
			break
		}
	}

	if channelID == "" {
		s.ChannelMessageSend(m.ChannelID, "You aren't in a voice channel")

		return
	}

	vc, err := s.ChannelVoiceJoin(guild.ID, channelID, false, true)

	if err != nil {
		fmt.Println("Failed to join voice channel: ", err)

		return
	}

	// Sleep for a specified amount of time before playing the sound
	time.Sleep(250 * time.Millisecond)

	// Start speaking.
	_ = vc.Speaking(true)

	//// Send the buffer data.
	//for i := 0; i < 10000; i++ {
	//	//vc.OpusSend <- []byte{127, 127, 127, 127, 127, 127, 127, 127, 127, 127, 127}
	//}

	// Stop speaking
	_ = vc.Speaking(false)

	// Sleep for a specificed amount of time before ending.
	time.Sleep(250 * time.Millisecond)

	// Disconnect from the provided voice channel.
	_ = vc.Disconnect()

	return
}

// Package shoutlikeart calculates whether you shout like art or not
package shoutlikeart

import (
	"github.com/bwmarrin/discordgo"
	"log"
)

type ShoutLikeArt struct {}

func (sla *ShoutLikeArt) Subscribe(dg discordgo.Session) {
	dg.AddHandler(sla.MessageCreate)
}

func (sla *ShoutLikeArt) MessageCreate(s discordgo.Session, m discordgo.MessageCreate) {
	if m.Content != "!sla" {
		return
	}

	channel, err := s.Channel(m.ChannelID)

	if err != nil {
		log.Println("Unable to get channel info: ", err)

		return
	}

	guild, err := s.Guild(channel.GuildID)

	if err != nil {
		log.Println("Unable to get guild info: ", err)

		return
	}

	var voiceChannel discordgo.Channel

	for _, voiceChannel = range guild.Channels {
		break
	}

	_, err = s.ChannelVoiceJoin(guild.ID, voiceChannel.ID, false, true)

	if err != nil {
		log.Println(err)

		return
	}
}

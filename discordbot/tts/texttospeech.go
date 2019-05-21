package tts

import (
	"context"
	"crypto/md5"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"golang.org/x/sync/errgroup"
	"io"
	"log"
	"unicode/utf8"
)

type TTSProvider interface {
	Process(text string) (io.Reader, error)
}

// TextToSpeech is a plugin to support text to speech feature
type TextToSpeech struct {
	Provider TTSProvider
}

func NewTTS(p TTSProvider) *TextToSpeech {
	return &TextToSpeech{
		Provider: p,
	}
}

// MessageReactionAdd
func(tts *TextToSpeech) MessageReactionAdd(s *discordgo.Session, mr *discordgo.MessageReactionAdd) {
	if mr.Emoji.Name != "🔈" {
		return
	}

	message, err := s.ChannelMessage(mr.ChannelID, mr.MessageID)
	if err != nil {
		log.Println("Error getting message: ", err)
		return
	}

	parts := splitTextToParts(message.Content, 255)
	processedParts := make([]io.Reader, len(parts))

	gr, _ := errgroup.WithContext(context.Background())
	for i, v := range parts {
		v := v
		i := i
		gr.Go(func() error {
			content, err := tts.Provider.Process(v)
			if err != nil {
				return err
			}
			processedParts[i] = content
			return nil
		})
	}
	if err := gr.Wait(); err != nil {
		s.ChannelMessageSend(mr.ChannelID, err.Error())
		log.Println("Error converting message to speech: ", err)
		return
	}

	mergedStream, err := MergeOggStreams(processedParts)
	if err != nil {
		log.Println("Error converting message to speech: ", err)
		return
	}

	s.ChannelMessageSendComplex(mr.ChannelID, &discordgo.MessageSend{
		Files:   []*discordgo.File{
			{
				Name:        fmt.Sprintf("%x", md5.Sum([]byte(message.Content))) + ".ogg",
				ContentType: "audio/ogg",
				Reader:      mergedStream,
			},
		},
	})
}

func splitTextToParts(text string, partLen int) []string {
	textLen := utf8.RuneCountInString(text)
	if textLen < partLen {
		return []string{text}
	}

	parts := []string{}
	index := 0
	procText := []rune(text)
	for index+partLen <= textLen {
		parts = append(parts, string(procText[index:index+partLen]))
		index += partLen
	}
	if index < textLen {
		parts = append(parts, string(procText[index:textLen]))
	}

	return parts
}
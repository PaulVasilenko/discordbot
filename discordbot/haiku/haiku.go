package haiku

import (
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const (
	eng = iota
	rus
	nothing
)

var rusVowels = regexp.MustCompile(`(?i)[аеёиоуыэюя]`)
var engLetters = regexp.MustCompile(`(?i)[a-z]`)
var engVowels = `aeiouy`

type Haiku struct{}

func NewHaiku() *Haiku {
	return &Haiku{}
}

// GetInfo returns map of info message
func (c *Haiku) GetInfo() map[string]string {
	return map[string]string{
		"Haiku": `Haiku automatically scans messages to figure out whether you can make haiku of them or not`,
	}
}

// MessageCreate reacts for created messages and processes logic
func (c *Haiku) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	haiku := makeHaiku(strings.FieldsFunc(m.Content, func(r rune) bool { return (r == ' ') || (r == '\n') }))
	if haiku == nil {
		return
	}

	messageContent := "🇯🇵🇯🇵🇯🇵🇯🇵🇯🇵🇯🇵\n"

	for i, row := range haiku {
		messageContent += strings.Join(row, " ")
		if i != len(haiku) {
			messageContent += "\n"
		}
	}
	messageContent += "🇯🇵🇯🇵🇯🇵🇯🇵🇯🇵🇯🇵"

	s.ChannelMessageSend(m.ChannelID, messageContent)
	return
}

func makeHaiku(words []string) [][]string {
	wordSyllable := make([]int, len(words))

	for i, word := range words {
		switch lang(word) {
		case eng:
			wordSyllable[i] = countSyllablesEng(word)
		case rus:
			wordSyllable[i] = countSyllablesRus(word)
		default:
			wordSyllable[i] = 0
		}
	}

	haiku := make([][]string, 3)

	// we now build only 5-7-5 haiku
	pattern := []int{5, 7, 5}

	row := 0
	success := false
	for i, word := range words {
		pattern[row] -= wordSyllable[i]
		// If word doesn't fit row there is no haiku
		if pattern[row] < 0 {
			return nil
		}

		haiku[row] = append(haiku[row], word)

		success = row == 2 && pattern[row] == 0
		if pattern[row] == 0 {
			row++
		}

		// if rows more then 3 there is no haiku
		if (row > 2) && (i+1 != len(words)) {
			return nil
		}
	}
	if !success {
		return nil
	}

	return haiku
}

func lang(text string) int {
	stat := struct {
		cyr   int
		latin int
	}{}
	for _, r := range []rune(text) {
		if r <= 'я' && r >= 'А' {
			stat.cyr++
		} else if r <= 'z' && r >= 'A' {
			stat.latin++
		}
	}

	if stat.cyr == 0 {
		return eng
	} else if stat.latin == 0 {
		return rus
	}

	return nothing
}

func countSyllablesRus(word string) int {
	// Fetching all vowels. All syllables contain only 1 vowel
	wordVowels := rusVowels.FindAllString(word, -1)
	if len(wordVowels) == 0 {
		return 1
	}
	return len(wordVowels)
}

func countSyllablesEng(word string) int {
	word = string(strings.Join(engLetters.FindAllString(word, -1), ""))
	count := 0
	for i, v := range word {
		if i == 0 {
			if contains(engVowels, v) {
				count++
			}
			continue
		}

		if contains(engVowels, v) && !contains(engVowels, rune(word[i-1])) {
			count++
		}
	}

	if strings.HasSuffix(word, "e") {
		count--
	}

	if strings.HasSuffix(word, "le") && len(word) > 2 && !contains(engVowels, rune(word[len(word)-4])) {
		count++
	}

	if count == 0 {
		count++
	}
	return count
}

func contains(src string, search rune) bool {
	for _, v := range src {
		if v == search {
			return true
		}
	}
	return false
}

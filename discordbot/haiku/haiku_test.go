package haiku

import (
	"fmt"
	"strings"
	"testing"
)

func Test_countSyllablesEng(t *testing.T) {
	type test struct {
		word          string
		syllableCount int
	}

	testCases := []test{
		{
			word:          "test",
			syllableCount: 1,
		},
		{
			word:          "alley",
			syllableCount: 2,
		},
		{
			word:          "robot",
			syllableCount: 2,
		},
		{
			word:          "seven",
			syllableCount: 2,
		},
		{
			word:          "castle",
			syllableCount: 2,
		},
		{
			word:          "thought",
			syllableCount: 1,
		},
	}

	for _, v := range testCases {
		actual := countSyllablesEng(v.word)
		if actual != v.syllableCount {
			fmt.Println("test failed with word: ", v.word)
			t.Error("expected:", v.syllableCount, "actual:", actual)
		}
	}
}

func Test_makeHaiku(t *testing.T) {
	type test struct {
		words []string
		haiku [][]string
	}

	testCases := []test{
		{
			words: strings.FieldsFunc(`An old silent pond
A frog jumps into the pond
Splash! Silence again.`, func(r rune) bool { return (r == ' ') || (r == '\n') }),
			haiku: [][]string{
				{"An", "old", "silent", "pond"},
				{"A", "frog", "jumps", "into", "the", "pond"},
				{"Splash!", "Silence", "again."},
			},
		},
		{
			words: strings.FieldsFunc(`Всё глазел на них,
Сакуры цветы, пока
Шею не свело`, func(r rune) bool { return (r == ' ') || (r == '\n') }),
			haiku: [][]string{
				{"Всё", "глазел", "на", "них,"},
				{"Сакуры", "цветы,", "пока"},
				{"Шею", "не", "свело"},
			},
		},
		{
			words: strings.FieldsFunc(`привет пока`, func(r rune) bool { return (r == ' ') || (r == '\n') }),
			haiku: nil,
		},
	}

	for _, v := range testCases {
		actual := makeHaiku(v.words)
		if actual == nil && v.haiku != nil {
			t.Error("nil result returned when not expected")
		}
		for i := range actual {
			for j := range actual[i] {
				if actual[i][j] != v.haiku[i][j] {
					fmt.Println("test failed with poem: ", v.words)
					t.Error("expected:", v.haiku, "actual:", actual)
					t.FailNow()
				}
			}
		}
	}
}

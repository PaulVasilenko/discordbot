package gamehighlighter

import (
	"errors"
	"github.com/bwmarrin/discordgo"
	_ "gopkg.in/mgo.v2"
	"strings"
)

const (
	GameLeagueOfLegends string = "lol"
	GameOverwatch       string = "ow"
)

// Confify is struct which represents Confify plugin with it's configurations
type GameHighlighter struct {
}

// Params is struct which represents string params passed to
type commandParameters struct {
	command string
	game    string
}

func NewGameHighlighter() *GameHighlighter {
	return &GameHighlighter{}
}

func (ghl *GameHighlighter) Subscribe(s *discordgo.Session) {
	s.AddHandler(ghl.MessageCreate)
}

func (ghl *GameHighlighter) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	params, err := ghl.parseMessage(m)

	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, err.Error())
		return
	}

	var res string

	switch params.command {
	case "!subscribegame":
	case "!subg":
		res, err = ghl.subscribe(params)
	case "!unsubscribegame":
	case "!unsubg":
		res, err = ghl.unsubscribe(params)
	case "!startmatch":
	case "!startm":
		res, err = ghl.startMatch(params)
	case "!stopmatch":
	case "!stopm":
		res, err = ghl.stopMatch(params)
	case "!joingame":
	case "!jg":
		res, err = ghl.joinGame(params)
	case "!leavegame":
	case "!lg":
		res, err = ghl.leaveGame(params)
	case "!currentplayers":
	case "!cp":
		res, err = ghl.currentPlayers(params)
	case "!highlightforlobby":
	case "!hfl":
		res, err = ghl.highlightForLobby(params)
	default:
		_, _ = s.ChannelMessageSend(m.ChannelID, "Bad command name")
		return
	}

	_, _ = s.ChannelMessageSend(m.ChannelID, res)
}

func (ght *GameHighlighter) parseMessage(m *discordgo.MessageCreate) (*commandParameters, error) {
	args := strings.Split(m.Content, " ")
	games := [2]string{GameLeagueOfLegends, GameOverwatch}
	inArray := false

	for _, v := range games {
		if inArray = v == args[1]; inArray {
			break
		}
	}

	if !inArray {
		return nil, errors.New("Unsupported game type")
	}

	return &commandParameters{command: args[0], game: args[1]}, nil
}

func (ghl *GameHighlighter) subscribe(params *commandParameters) (string, error) {
	return "", nil
}

func (ghl *GameHighlighter) unsubscribe(params *commandParameters) (string, error) {
	return "", nil
}

func (ghl *GameHighlighter) startMatch(params *commandParameters) (string, error) {
	return "", nil
}

func (ghl *GameHighlighter) stopMatch(params *commandParameters) (string, error) {
	return "", nil
}

func (ghl *GameHighlighter) joinGame(params *commandParameters) (string, error) {
	return "", nil
}

func (ghl *GameHighlighter) leaveGame(params *commandParameters) (string, error) {
	return "", nil
}

func (ghl *GameHighlighter) currentPlayers(params *commandParameters) (string, error) {
	return "", nil
}

func (ghl *GameHighlighter) highlightForLobby(params *commandParameters) (string, error) {
	return "", nil
}

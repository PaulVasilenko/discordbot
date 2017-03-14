package gamehighlighter

import (
	"errors"
	"flag"
	"github.com/bwmarrin/discordgo"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	_ "gopkg.in/mgo.v2/bson"
	"strings"
)

const (
	GameLeagueOfLegends        string = "lol"
	GameLeagueOfLegendsVerbose string = "League Of Legends"
	GameOverwatch              string = "ow"
	GameOverwatchVerbose       string = "Overwatch"

	CollectionSession string = "Session"

	CommandSubscribeGame        string = "!subscribegame"
	CommandSubscribeGameShort   string = "!subg"
	CommandUnsubscribeGame      string = "!unsubscribegame"
	CommandUnsubscribeGameShort string = "!unsubg"
	CommandStartSession         string = "!startsession"
	CommandStartSessionShort    string = "!starts"
	CommandStopSession          string = "!stopsession"
	CommandStopSessionShort     string = "!stops"
	CommandJoinSession          string = "!joingame"
	CommandJoinSessionShort     string = "!jg"
	CommandLeaveSession         string = "!leavegame"
	CommandLeaveSessionShort    string = "!lg"
	CommandCurrentPlayers       string = "!currentplayers"
	CommandCurrentPlayersShort  string = "!cp"
	CommandLobbyHighlight       string = "!highlightforlobby"
	CommandLobbyHighlightShort  string = "!hfl"

	CommandNotInLobbyHighlight      string = "!highlightnotinlobby"
	CommandNotInLobbyHighlightShort string = "!hfnl"
)

var (
	MongoDbHost = flag.String("mongoDbHost", "127.0.0.1", "Mongo Db Host")
	MongoDbPort = flag.String("mongoDbPort", "27017", "Mongo Db Port")
)

// Confify is struct which represents Confify plugin with it's configurations
type GameHighlighter struct {
	mongoDbConn *mgo.Session
}

type User struct {
	ID       string `bson:"id"`
	Username string `bson:"username"`
}

type Session struct {
	ID          bson.ObjectId `bson:"_id,omitempty"`
	IsStarted   bool          `bson:"isstarted"`
	Subscribers []User        `bson:"Subscribers"`
	Joined      []User        `bson:"Joined"`
}

// Params is struct which represents params passed to functions
type commandParameters struct {
	command     string
	game        string
	gameVerbose string
	author      *discordgo.User
}

func NewGameHighlighter() (*GameHighlighter, error) {
	session, err := mgo.Dial("mongodb://" + *MongoDbHost + ":" + *MongoDbPort)

	collectionSubsIndex := mgo.Index{
		Key:      []string{"Subscribers.ID"},
		Unique:   true,
		DropDups: true}

	collectionJoinedIndex := mgo.Index{
		Key:      []string{"Joined.ID"},
		Unique:   true,
		DropDups: true}

	for _, v := range [2]string{GameLeagueOfLegends, GameOverwatch} {
		sessionCount, err := session.DB(v).C(CollectionSession).Find(bson.M{}).Count()

		session.DB(v).C(CollectionSession).EnsureIndex(collectionSubsIndex)
		session.DB(v).C(CollectionSession).EnsureIndex(collectionJoinedIndex)

		if err != nil {
			return nil, err
		}

		if sessionCount < 1 {
			if err := session.DB(v).C(CollectionSession).Insert(&Session{IsStarted: false}); err != nil {
				return nil, err
			}
		}
	}

	if err != nil {
		return nil, err
	}

	return &GameHighlighter{mongoDbConn: session}, nil
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

	if params == nil {
		return
	}

	var res string

	switch params.command {
	case CommandSubscribeGame:
	case CommandSubscribeGameShort:
		res, err = ghl.subscribe(params)
	case CommandUnsubscribeGame:
	case CommandUnsubscribeGameShort:
		res, err = ghl.unsubscribe(params)
	case CommandStartSession:
	case CommandStartSessionShort:
		res, err = ghl.startSession(params)
	case CommandStopSession:
	case CommandStopSessionShort:
		res, err = ghl.stopSession(params)
	case CommandJoinSession:
	case CommandJoinSessionShort:
		res, err = ghl.joinSession(params)
	case CommandLeaveSession:
	case CommandLeaveSessionShort:
		res, err = ghl.leaveSession(params)
	case CommandCurrentPlayers:
	case CommandCurrentPlayersShort:
		res, err = ghl.currentPlayers(params)
	case CommandLobbyHighlight:
	case CommandLobbyHighlightShort:
		res, err = ghl.highlightForLobby(params)
	case CommandNotInLobbyHighlight:
	case CommandNotInLobbyHighlightShort:
		res, err = ghl.highlightNotInLobby(params)
	default:
		return
	}

	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Error occured, notify PandaSam about it: "+err.Error())
	}

	_, _ = s.ChannelMessageSend(m.ChannelID, res)
}

func (ght *GameHighlighter) parseMessage(m *discordgo.MessageCreate) (*commandParameters, error) {
	args := strings.Split(m.Content, " ")

	inArray := false

	commands := [18]string{
		CommandSubscribeGame,
		CommandSubscribeGameShort,
		CommandUnsubscribeGame,
		CommandUnsubscribeGameShort,
		CommandStartSession,
		CommandStartSessionShort,
		CommandStopSession,
		CommandStopSessionShort,
		CommandJoinSession,
		CommandJoinSessionShort,
		CommandLeaveSession,
		CommandLeaveSessionShort,
		CommandCurrentPlayers,
		CommandCurrentPlayersShort,
		CommandLobbyHighlight,
		CommandLobbyHighlightShort,
		CommandNotInLobbyHighlight,
		CommandNotInLobbyHighlightShort}

	for _, v := range commands {
		if inArray = v == args[0]; inArray {
			break
		}
	}

	if !inArray {
		return nil, nil
	}

	inArray = false

	if len(args) < 2 {
		return nil, errors.New("Game code required")
	}

	games := map[string]string{
		GameLeagueOfLegends: GameLeagueOfLegendsVerbose,
		GameOverwatch:       GameOverwatchVerbose}

	for v := range games {
		if inArray = v == args[1]; inArray {
			break
		}
	}

	if !inArray {
		return nil, errors.New("Unsupported game code")
	}

	return &commandParameters{command: args[0], game: args[1], gameVerbose: games[args[1]], author: m.Author}, nil
}

func (ghl *GameHighlighter) subscribe(params *commandParameters) (string, error) {
	err := ghl.mongoDbConn.DB(params.game).C(CollectionSession).Update(
		bson.M{},
		bson.M{"$addToSet": bson.M{"Subscribers": params.author}})

	if err != nil {
		return "", err
	}

	return "<@" + params.author.ID + "> subscribed to " + params.gameVerbose, nil
}

func (ghl *GameHighlighter) unsubscribe(params *commandParameters) (string, error) {
	err := ghl.mongoDbConn.DB(params.game).C(CollectionSession).Update(
		bson.M{},
		bson.M{"$pull": bson.M{"Subscribers": params.author}})

	if err != nil {
		return "", err
	}

	return "<@" + params.author.ID + "> unsubscribed from " + params.gameVerbose, nil
}

func (ghl *GameHighlighter) startSession(params *commandParameters) (string, error) {
	var session *Session

	if err := ghl.mongoDbConn.DB(params.game).C(CollectionSession).Find(bson.M{}).One(&session); err != nil {
		return "", err
	}

	if session.IsStarted {
		return "Session is already started", nil
	}

	err := ghl.mongoDbConn.DB(params.game).C(CollectionSession).Update(
		bson.M{},
		bson.M{"$set": bson.M{"Joined": []string{}, "isstarted": true}})

	if err != nil {
		return "", err
	}

	ids := []string{}

	for _, v := range session.Subscribers {
		ids = append(ids, v.ID)
	}

	return params.gameVerbose + " session started, highlighting subscribers: <@" + strings.Join(ids, ">, <@") + ">", nil
}

func (ghl *GameHighlighter) stopSession(params *commandParameters) (string, error) {
	var session *Session

	if err := ghl.mongoDbConn.DB(params.game).C(CollectionSession).Find(bson.M{}).One(&session); err != nil {
		return "", err
	}

	if !session.IsStarted {
		return "Session isn't started", nil
	}

	err := ghl.mongoDbConn.DB(params.game).C(CollectionSession).Update(
		bson.M{},
		bson.M{"$set": bson.M{"Joined": []string{}, "isstarted": false}})

	if err != nil {
		return "", err
	}

	return params.gameVerbose + " session stopped", nil
}

func (ghl *GameHighlighter) joinSession(params *commandParameters) (string, error) {
	var session *Session

	if err := ghl.mongoDbConn.DB(params.game).C(CollectionSession).Find(bson.M{}).One(&session); err != nil {
		return "", err
	}

	if !session.IsStarted {
		return "Session isn't started, you cannot join", nil
	}

	err := ghl.mongoDbConn.DB(params.game).C(CollectionSession).Update(
		bson.M{},
		bson.M{"$addToSet": bson.M{"Joined": params.author}})

	if err != nil {
		return "", err
	}

	return "<@" + params.author.ID + "> joins to " + params.gameVerbose, nil
}

func (ghl *GameHighlighter) leaveSession(params *commandParameters) (string, error) {
	var session *Session

	if err := ghl.mongoDbConn.DB(params.game).C(CollectionSession).Find(bson.M{}).One(&session); err != nil {
		return "", err
	}

	if !session.IsStarted {
		return "Session isn't started", nil
	}

	err := ghl.mongoDbConn.DB(params.game).C(CollectionSession).Update(
		bson.M{},
		bson.M{"$pull": bson.M{"Joined": params.author}})

	if err != nil {
		return "", err
	}

	return "<@" + params.author.ID + "> leaves " + params.gameVerbose, nil
}

func (ghl *GameHighlighter) currentPlayers(params *commandParameters) (string, error) {
	var session *Session

	if err := ghl.mongoDbConn.DB(params.game).C(CollectionSession).Find(bson.M{}).One(&session); err != nil {
		return "", err
	}

	if !session.IsStarted {
		return "Session isn't started, no players", nil
	}

	usernames := []string{}

	for _, v := range session.Joined {
		usernames = append(usernames, v.Username)
	}

	return "Currently playing " + params.gameVerbose + ": " + strings.Join(usernames, ", "), nil
}

func (ghl *GameHighlighter) highlightForLobby(params *commandParameters) (string, error) {
	var session *Session

	ghl.mongoDbConn.DB(params.game).C(CollectionSession).Find(bson.M{}).One(&session)

	if !session.IsStarted {
		return "Session isn't started, not highlighting", nil
	}

	ids := []string{}

	for _, v := range session.Joined {
		ids = append(ids, v.ID)
	}

	return "Players highlighted are in " + params.gameVerbose + ": <@" + strings.Join(ids, ">, <@") + ">", nil
}

func (ghl *GameHighlighter) highlightNotInLobby(params *commandParameters) (string, error) {
	var session *Session

	ghl.mongoDbConn.DB(params.game).C(CollectionSession).Find(bson.M{}).One(&session)

	if !session.IsStarted {
		return "Session isn't started, not highlighting", nil
	}

	subscribedIDs := []string{}

	for _, v := range session.Subscribers {
		subscribedIDs = append(subscribedIDs, v.ID)
	}

	ids := []string{}

	for _, v := range session.Joined {
		ids = append(ids, v.ID)
	}

	intersectingValues := difference(subscribedIDs, ids)

	return "Players highlighted to join " + params.gameVerbose + ": <@" + strings.Join(intersectingValues, ">, <@") + ">", nil
}

func difference(slice1 []string, slice2 []string) []string {
	var diff []string

	// Loop two times, first to find slice1 strings not in slice2,
	// second loop to find slice2 strings not in slice1
	for i := 0; i < 2; i++ {
		for _, s1 := range slice1 {
			found := false
			for _, s2 := range slice2 {
				if s1 == s2 {
					found = true
					break
				}
			}
			// String not found. We add it to return slice
			if !found {
				diff = append(diff, s1)
			}
		}
		// Swap the slices, only if it was the first loop
		if i == 0 {
			slice1, slice2 = slice2, slice1
		}
	}

	return diff
}

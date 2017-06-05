package racing

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"math/rand"
	"strings"
	"time"
)

const (
	MongoCollectionRacing = "racing"
	MongoDBRacing         = "racing"

	CommandJoinRace   = "!rjoin"
	CommandLeaveRace  = "!rleave"
	CommandStartRace  = "!rstart"
	CommandResetRace  = "!rreset"
	CommandJoinedRace = "!rjoined"

	RacerEmoji      = ":wheelchair:"
	RacerCloudEmoji = ":cloud:"

	RaceDelimiter    = "   "
	RaceTrackLength  = 25
	SpeedCoefficient = 70
)

type Racer struct {
	ID       string `bson:"id"`
	Username string `bson:"username"`
}

type Racing struct {
	mongoDbConn *mgo.Session
}

func NewRacing(MongoDbHost, MongoDbPort string) (*Racing, error) {
	session, err := mgo.Dial("mongodb://" + MongoDbHost + ":" + MongoDbPort)

	if err != nil {
		return nil, err
	}

	collectionJoinedIndex := mgo.Index{
		Key:      []string{"id"},
		Unique:   true,
		DropDups: true}

	err = session.DB(MongoDBRacing).C(MongoCollectionRacing).EnsureIndex(collectionJoinedIndex)

	if err != nil {
		return nil, err
	}

	return &Racing{mongoDbConn: session}, nil
}

func (r *Racing) Subscribe(s *discordgo.Session) {
	s.AddHandler(r.MessageCreate)
}

func (q *Racing) GetInfo() map[string]string {
	return map[string]string{
		CommandResetRace:  "Removes all joined players to race",
		CommandStartRace:  "Starts race with all joined players",
		CommandJoinRace:   "Joins to race",
		CommandLeaveRace:  "Leaves race",
		CommandJoinedRace: "Prints list of joined racers",
	}
}

func (r *Racing) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	switch {
	case m.Content == CommandResetRace:
		r.reset(s, m)
	case m.Content == CommandStartRace:
		r.start(s, m)
	case m.Content == CommandJoinRace:
		r.join(s, m)
	case m.Content == CommandLeaveRace:
		r.leave(s, m)
	case m.Content == CommandJoinedRace:
		r.joined(s, m)
	}
}

// TODO: Upgrade this method to prevent network delays affecting race
func (r *Racing) start(s *discordgo.Session, m *discordgo.MessageCreate) {
	var racers []*Racer
	r.mongoDbConn.DB(MongoDBRacing).C(MongoCollectionRacing).Find(bson.M{}).All(&racers)

	var winners []*Racer
	ra := rand.New(rand.NewSource(time.Now().UnixNano()))
	coef := make([]int, len(racers))

	raceMessage, err := s.ChannelMessageSend(m.ChannelID, "Race is loading...")

	if err != nil {
		log.Println(err)
		return
	}

	racersMessage := racerPlaces(racers, coef)

	s.ChannelMessageEdit(m.ChannelID, raceMessage.ID, "Ready!\n"+racersMessage)

	for i := 0; i < 2; i++ {
		time.Sleep(1 * time.Second)
		switch i {
		case 0:
			s.ChannelMessageEdit(m.ChannelID, raceMessage.ID, "Steady!\n"+racersMessage)
		case 1:
			s.ChannelMessageEdit(m.ChannelID, raceMessage.ID, "GO!\n"+racersMessage)
		}
	}

	c := time.Tick(1300 * time.Millisecond)

	timer := 0

	for range c {
		timer += 1
		shouldStop := true

		// Calculate coefficients
		for i, racer := range racers {
			if coef[i] >= RaceTrackLength {
				continue
			}

			coef[i] += 1

			if coef[i] == RaceTrackLength {
				winners = append(winners, racer)
				continue
			}

			if ra.Intn(101) < SpeedCoefficient {
				coef[i] += 1
			}

			if coef[i] == RaceTrackLength {
				winners = append(winners, racer)
			}

			if coef[i] < RaceTrackLength {
				shouldStop = false
			}
		}

		message := racerPlaces(racers, coef)
		_, err := s.ChannelMessageEdit(
			m.ChannelID,
			raceMessage.ID,
			fmt.Sprintf(
				"%s %s  |Finish; Time: %02d:%02d\n%s",
				"GO!",
				strings.Repeat(RaceDelimiter, RaceTrackLength-1),
				timer/60,
				timer%60,
				message,
			),
		)

		if err != nil {
			log.Println("Race error: ", err)
		}

		if shouldStop {
			break
		}
	}

	message := "Race result:\n"

	for place, racer := range winners {
		message += fmt.Sprintf("#%d - %s\n", place+1, racer.Username)
	}

	s.ChannelMessageSend(m.ChannelID, message)
}

func (r *Racing) join(s *discordgo.Session, m *discordgo.MessageCreate) {
	err := r.mongoDbConn.DB(MongoDBRacing).C(MongoCollectionRacing).Insert(&Racer{ID: m.Author.ID, Username: m.Author.Username})

	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			s.ChannelMessageSend(m.ChannelID, "You have already joined to race")
			return
		}

		log.Println(err)
		return
	}

	s.ChannelMessageSend(m.ChannelID, m.Author.Username+" successfully joined to next race")
}

func (r *Racing) leave(s *discordgo.Session, m *discordgo.MessageCreate) {
	r.mongoDbConn.DB(MongoDBRacing).C(MongoCollectionRacing).Remove(&Racer{ID: m.Author.ID, Username: m.Author.Username})
	s.ChannelMessageSend(m.ChannelID, m.Author.Username+" successfully left")
}

func (r *Racing) reset(s *discordgo.Session, m *discordgo.MessageCreate) {
	r.mongoDbConn.DB(MongoDBRacing).C(MongoCollectionRacing).RemoveAll(bson.M{})
	s.ChannelMessageSend(m.ChannelID, "Resetted racing")
}

func (r *Racing) joined(s *discordgo.Session, m *discordgo.MessageCreate) {
	var racers []*Racer
	r.mongoDbConn.DB(MongoDBRacing).C(MongoCollectionRacing).Find(bson.M{}).All(&racers)

	message := ""

	for k, v := range racers {
		message += fmt.Sprintf("#%d - %s \n", k+1, v.Username)
	}

	s.ChannelMessageSend(m.ChannelID, "Racers:\n"+message)
}

func racerPlaces(racers []*Racer, coef []int) string {
	message := ""

	for k, v := range racers {
		place := strings.Repeat(RaceDelimiter, coef[k])
		message += fmt.Sprintf("%s%s-%s\n", place, RacerEmoji, v.Username)
	}

	return message
}

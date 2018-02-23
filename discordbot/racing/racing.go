package racing

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/patrickmn/go-cache"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"math/rand"
	"strings"
	"time"
	//"regexp"
	"context"
	"database/sql"
	"regexp"
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

	DatabasePandaBot = "pandabot"
	DatetimeLayout   = "2006-01-02 15:04:05"

	RaceDelimiter   = "   "
	RaceTrackLength = 25

	MaximumDiscordMessageLength = 2000

	SpeedCoefficient       = 70
	BeingSlowedCoefficient = 10
	SmileyRegex            = `(?i)(\:[\w\d\_]+\:(\:([\w\d]+\-)+[\w\d]+\:)?)`
)

type Racer struct {
	ID       string `bson:"id"`
	Username string `bson:"username"`
	Emoticon string `bson:"emoticon"`
}

type RacerStats struct {
	Racer      *Racer
	FinishTime int
}

type Racing struct {
	mysqlDbConn *sql.DB
	mongoDbConn *mgo.Session
	cache       *cache.Cache
}

func NewRacing(MongoDbHost, MongoDbPort, MysqlDbHost, MysqlDbPort, MysqlDbUser, MysqlDbPassword string) (*Racing, error) {
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

	c := cache.New(cache.NoExpiration, cache.NoExpiration)

	dsn := MysqlDbUser + ":" + MysqlDbPassword + "@tcp(" + MysqlDbHost + ":" + MysqlDbPort + ")/" + DatabasePandaBot
	db, err := sql.Open("mysql", dsn)

	if err != nil {
		return nil, err
	}

	return &Racing{mongoDbConn: session, cache: c, mysqlDbConn: db}, nil
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
	case strings.HasPrefix(m.Content, CommandJoinRace):
		r.join(s, m)
	case m.Content == CommandLeaveRace:
		r.leave(s, m)
	case m.Content == CommandJoinedRace:
		r.joined(s, m)
	}
}

// TODO: Upgrade this method to prevent network delays affecting race
// TODO: Refactor
func (r *Racing) start(s *discordgo.Session, m *discordgo.MessageCreate) {
	if _, ok := r.cache.Get("Racing"); ok {
		s.ChannelMessageSend(m.ChannelID, "Race is already started somewhere. Please, wait until it ends")
		return
	}

	r.cache.Set("Racing", true, cache.NoExpiration)
	defer r.cache.Delete("Racing")

	var racers []*Racer
	r.mongoDbConn.DB(MongoDBRacing).C(MongoCollectionRacing).Find(bson.M{}).All(&racers)

	var winners []*RacerStats
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

			if ra.Intn(101) < BeingSlowedCoefficient {
				shouldStop = false
				continue
			}

			coef[i] += 1

			if coef[i] == RaceTrackLength {
				winners = append(winners, &RacerStats{Racer: racer, FinishTime: timer})
				continue
			}

			if ra.Intn(101) < SpeedCoefficient {
				coef[i] += 1
			}

			if coef[i] == RaceTrackLength {
				winners = append(winners, &RacerStats{Racer: racer, FinishTime: timer})
				continue
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
				"%s %s |Finish; Time: %02d:%02d\n%s",
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

	place := 0
	previousTime := 0

	for _, w := range winners {
		if previousTime != w.FinishTime {
			place += 1
		}
		previousTime = w.FinishTime
		message += fmt.Sprintf("**#%d - %s;** Finish Time: %d s; Speed: %f m/s;\n", place, w.Racer.Username, w.FinishTime, float64(RaceTrackLength)/float64(w.FinishTime))
	}

	go r.saveRacerStats(winners)

	s.ChannelMessageSend(m.ChannelID, message)
}

func (r *Racing) join(s *discordgo.Session, m *discordgo.MessageCreate) {
	count, err := r.mongoDbConn.DB(MongoDBRacing).C(MongoCollectionRacing).Find(bson.M{}).Count()

	if err != nil {
		log.Println(err)
		return
	}

	// Assuming we're saving this user
	count += 1

	// Calculate maximum number of symbols might be used to represent race
	// 35 is maximum username length + several symbols
	// 22 is number of not calculated symbols used
	symbolsThreshold := count*(RaceTrackLength*len(RaceDelimiter)+35) + len(RaceDelimiter)*RaceTrackLength + 22

	if symbolsThreshold >= MaximumDiscordMessageLength {
		s.ChannelMessageSend(m.ChannelID, "You cannot join race: maximum number of racers are already joined")
		return
	}

	racer := &Racer{ID: m.Author.ID, Username: m.Author.Username, Emoticon: RacerEmoji}

	emojiRegex := regexp.MustCompile(SmileyRegex)

	smiley := emojiRegex.FindString(m.Content)
	fmt.Println(m.Content)

	if smiley != "" {
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

		racer.Emoticon = smiley

		fmt.Println(smiley)
		fmt.Println(racer.Emoticon)

		for _, emoji := range guild.Emojis {
			if smiley == (":" + emoji.Name + ":") {
				racer.Emoticon = "<" + smiley + emoji.ID + ">"

				break
			}
		}

		fmt.Println(racer.Emoticon)
	}

	err = r.mongoDbConn.DB(MongoDBRacing).C(MongoCollectionRacing).Insert(racer)

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
	r.mongoDbConn.DB(MongoDBRacing).C(MongoCollectionRacing).Remove(bson.M{"id": m.Author.ID})
	s.ChannelMessageSend(m.ChannelID, m.Author.Username+" successfully left next race")
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

func (r *Racing) saveRacerStats(winners []*RacerStats) {
	tx, err := r.mysqlDbConn.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false})

	if err != nil {
		log.Println(err)
		return
	}

	if r, err := tx.Query(`INSERT INTO raceHistory (raceDatetime) VALUES (?);`, time.Now().Format(DatetimeLayout)); err != nil {
		r.Close()
		log.Println(err)
		return
	} else {
		r.Close()
	}

	var raceId int

	if err := tx.QueryRow("SELECT LAST_INSERT_ID();").Scan(&raceId); err != nil {
		log.Println(err)
		tx.Rollback()
		return
	}

	query := `
		INSERT INTO raceHistoryStats
			(raceId, racerId, racerUsername, racerSpeed, racerTime, place)
		VALUES`

	values := make([]string, len(winners))

	for i, racer := range winners {
		values[i] = fmt.Sprintf(
			"(%d, '%s', '%s', %f, %d, %d)",
			raceId,
			racer.Racer.ID,
			racer.Racer.Username,
			float64(RaceTrackLength)/float64(racer.FinishTime),
			racer.FinishTime,
			i,
		)
	}

	query += strings.Join(values, ",")

	if r, err := tx.Query(query + ";"); err != nil {
		r.Close()
		log.Println(err)
		tx.Rollback()
		return
	} else {
		r.Close()
	}

	tx.Commit()
}

func racerPlaces(racers []*Racer, coef []int) string {
	message := ""

	for k, v := range racers {
		place := strings.Repeat(RaceDelimiter, coef[k])
		racerEmoji := RacerEmoji
		if v.Emoticon != "" {
			racerEmoji = v.Emoticon
		}

		message += fmt.Sprintf("%s%s-%s\n", place, racerEmoji, v.Username)
	}

	return message
}

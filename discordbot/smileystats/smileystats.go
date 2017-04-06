// package smileystats provides plugin to calculate usage statistics of emoticons
package smileystats

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"regexp"
)

const (
	MongoDatabaseSmileyStats   string = "smileystats"
	MongoCollectionSmileyStats string = "smileystats"
	SmileyRegex                string = `(?i)(\:[\w\d\_]+\:(\:([\w\d]+\-)+[\w\d]+\:)?)`
)

// SmileyStats is struct which represents plugin configuration
type SmileyStats struct {
	mongoDbConn *mgo.Session
}

// Smiley is a struct which represents MongoDB schema of smiley
type Smiley struct {
	ID       bson.ObjectId `bson:"_id,omitempty"`
	Emoticon *Emoji        `bson:"smiley"`
	Count    int           `bson:"count,omitempty"`
}

// Emoji contains emoji-related info
type Emoji struct {
	Name string `bson:"name"`
	ID   string `bson:"id,omitempty"`
}

// NewSmileyStats returns set up instance of SmileyStats
func NewSmileyStats(MongoDbHost, MongoDbPort string) (*SmileyStats, error) {
	session, err := mgo.Dial("mongodb://" + MongoDbHost + ":" + MongoDbPort)

	if err != nil {
		return nil, err
	}

	smileyUniqueIndex := mgo.Index{
		Key:      []string{"smiley.name"},
		Unique:   true,
		DropDups: true}

	session.DB(MongoDatabaseSmileyStats).C(MongoCollectionSmileyStats).EnsureIndex(smileyUniqueIndex)

	return &SmileyStats{mongoDbConn: session}, nil
}

// Subscribe is method which subscribes plugin to all needed events
func (sm *SmileyStats) Subscribe(dg *discordgo.Session) {
	dg.AddHandler(sm.MessageCreate)
}

// MessageCreate is method which triggers when message sent to discord chat
func (sm *SmileyStats) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Content == "!printtopsmileys" || m.Content == "!pts" {
		sm.printTopStats(s, m.ChannelID)

		return
	}

	if m.Author.Bot {
		return
	}

	regexpSmiley, err := regexp.Compile(SmileyRegex)

	if err != nil {
		log.Println(err)

		return
	}

	smileys := regexpSmiley.FindAllString(m.Content, -1)

	if smileys == nil {
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

	coll := sm.mongoDbConn.DB(MongoDatabaseSmileyStats).C(MongoCollectionSmileyStats)

	// Server specific IDs
	// TODO: Improve speed of algorithm
	for _, emoji := range guild.Emojis {
		idsToRemove := []int{}

		for i, smiley := range smileys {
			if smiley == (":" + emoji.Name + ":") {
				smileyStruct := &Smiley{Emoticon: &Emoji{Name: smiley, ID: emoji.ID}, Count: 0}

				coll.Insert(smileyStruct)
				coll.Update(smileyStruct, bson.M{"$inc": bson.M{"count": 1}})

				idsToRemove = append(idsToRemove, i)
			}
		}

		for _, i := range idsToRemove {
			if len(smileys) == 1 {
				smileys = []string{}
			} else {
				smileys[i] = smileys[len(smileys)-1]
			}
		}
	}

	// Common ids
	for _, smiley := range smileys {
		smileyStruct := &Smiley{Emoticon: &Emoji{Name: smiley, ID: ""}, Count: 0}

		coll.Insert(smileyStruct)
		coll.Update(smileyStruct, bson.M{"$inc": bson.M{"count": 1}})
	}
}

func (sm *SmileyStats) printTopStats(s *discordgo.Session, channelID string) {
	var topSmileys []Smiley

	sm.mongoDbConn.DB(MongoDatabaseSmileyStats).C(MongoCollectionSmileyStats).
		Find(bson.M{}).
		Sort("-count").
		Limit(10).
		All(&topSmileys)

	stats := "Smileys top:\n "

	for i, v := range topSmileys {
		smileyString := ""

		if v.Emoticon.ID != "" {
			smileyString = fmt.Sprintf("<%s%v>", v.Emoticon.Name, v.Emoticon.ID)
		} else {
			smileyString = v.Emoticon.Name
		}

		stats += fmt.Sprintf("#%d - %s %d usages\n", i+1, smileyString, v.Count)
	}

	s.ChannelMessageSend(channelID, stats)
}

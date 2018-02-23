// package smileystats provides plugin to calculate usage statistics of emoticons
package smileystats

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"github.com/bwmarrin/discordgo"
	_ "github.com/go-sql-driver/mysql"
	"github.com/patrickmn/go-cache"
	"log"
	"regexp"
	"strings"
	"time"
)

const (
	DatabaseSmileyStats string = "pandabot"
	SmileyRegex         string = `(?i)(\:[\w\d\_]+\:(\:([\w\d]+\-)+[\w\d]+\:)?)`
)

// SmileyStats is struct which represents plugin configuration
type SmileyStats struct {
	dbConn *sql.DB
	cache  *cache.Cache
}

// NewSmileyStats returns set up instance of SmileyStats
func NewSmileyStats(MysqlDbHost, MysqlDbPort, MysqlDbUser, MysqlDbPassword string) (*SmileyStats, error) {
	dsn := MysqlDbUser + ":" + MysqlDbPassword + "@tcp(" + MysqlDbHost + ":" + MysqlDbPort + ")/" + DatabaseSmileyStats
	db, err := sql.Open("mysql", dsn)
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(50)
	db.SetConnMaxLifetime(1 * time.Second)

	if err != nil {
		return nil, err
	}

	c := cache.New(5*time.Second, 10*time.Second)

	return &SmileyStats{dbConn: db, cache: c}, nil
}

// Subscribe is method which subscribes plugin to all needed events
func (sm *SmileyStats) Subscribe(dg *discordgo.Session) {
	dg.AddHandler(sm.MessageCreate)
}

func (sm *SmileyStats) GetInfo() map[string]string {
	return map[string]string{
		"!pts": "Prints top 10 of amojis used. Pass emoji as an argument to see personal stat for this emoji",
	}
}

// MessageCreate is method which triggers when message sent to discord chat
func (sm *SmileyStats) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	if m.Content == "!printtopsmileys" || m.Content == "!pts" {
		if err := sm.printTopStats(s, m.ChannelID); err != nil {
			log.Println("printTopStats error: ", err)
		}

		return
	}

	regexpSmiley, err := regexp.Compile(SmileyRegex)

	if err != nil {
		log.Println(err)

		return
	}

	smileys := regexpSmiley.FindAllString(m.Content, -1)

	if strings.HasPrefix(m.Content, "!pts") {
		sm.printSmileyStat(s, smileys[0], m.ChannelID)

		return
	}

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

	// Server specific IDs
	// TODO: Improve speed of algorithm
	for i, smiley := range smileys {
		idsToRemove := []int{}
		hash := md5.Sum([]byte("Smiley_" + m.Author.Username + smiley))

		if _, ok := sm.cache.Get(string(hash[:])); ok == true {
			continue
		}

		for _, emoji := range guild.Emojis {
			if smiley == (":" + emoji.Name + ":") {
				if err := sm.insertSmiley(emoji.ID, smiley, m.Author.ID, m.Author.Username); err != nil {
					log.Println("Smiley Insert Failed: ", err)

					return
				}
				sm.cache.SetDefault(string(hash[:]), string(hash[:]))
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
		hash := md5.Sum([]byte("Smiley_" + m.Author.Username + smiley))

		if _, ok := sm.cache.Get(string(hash[:])); ok == true {
			continue
		}

		if err := sm.insertSmiley("", smiley, m.Author.ID, m.Author.Username); err != nil {
			log.Println("Smiley Insert Failed: ", err)

			return
		}

		sm.cache.SetDefault(string(hash[:]), string(hash[:]))
	}
}

func (sm *SmileyStats) insertSmiley(emojiID, emojiName, authorID, authorName string) error {
	sqlString := `
		INSERT IGNORE INTO smileyHistory
			(emojiId, emojiName, userId, userName, createDatetime)
		VALUES
			(?, ?, ?, ?, ?);`

	r, err := sm.dbConn.Query(
		sqlString,
		emojiID,
		emojiName,
		authorID,
		authorName,
		time.Now().Format("2006-01-02 15:04:05"),
	)

	if err != nil {
		return err
	}

	defer r.Close()

	return nil
}

func (sm *SmileyStats) printTopStats(s *discordgo.Session, channelID string) error {
	sqlString := `
	SELECT COUNT(emojiId) as usages, emojiName, emojiId
	FROM smileyHistory
	GROUP BY emojiName ORDER BY usages DESC LIMIT 10`

	rows, err := sm.dbConn.Query(sqlString)

	if err != nil {
		return err
	}

	defer rows.Close()

	stats := "Smileys top:\n"

	i := 0
	for rows.Next() {
		i += 1

		var count, emoticonName, emoticonId string
		rows.Scan(&count, &emoticonName, &emoticonId)

		smileyString := ""

		if emoticonId != "" {
			smileyString = fmt.Sprintf("<%s%v>", emoticonName, emoticonId)
		} else {
			smileyString = emoticonName
		}

		stats += fmt.Sprintf("#%d - %s %s usages\n", i, smileyString, count)
	}

	s.ChannelMessageSend(channelID, stats)

	return nil
}

func (sm *SmileyStats) printSmileyStat(s *discordgo.Session, smiley, channelID string) error {
	sqlString := `
	SELECT COUNT(emojiId) as usages, emojiName, emojiId, userName
	FROM smileyHistory
	WHERE emojiName = ?
	GROUP BY userName ORDER BY usages DESC LIMIT 10`

	rows, err := sm.dbConn.Query(sqlString, smiley)

	if err != nil {
		return err
	}

	defer rows.Close()

	stats := ""

	i := 0
	for rows.Next() {
		i += 1

		var count, emoticonName, emoticonId, userName string
		rows.Scan(&count, &emoticonName, &emoticonId, &userName)

		if i == 1 {
			smileyString := ""

			if emoticonId != "" {
				smileyString = fmt.Sprintf("<%s%v>", emoticonName, emoticonId)
			} else {
				smileyString = emoticonName
			}

			stats += fmt.Sprintf("Smiley %s top:\n", smileyString)
		}

		stats += fmt.Sprintf("#%d - %s %s usages\n", i, userName, count)
	}

	s.ChannelMessageSend(channelID, stats)

	return nil
}

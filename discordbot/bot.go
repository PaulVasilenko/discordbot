package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/facebookgo/flagconfig"
	"github.com/paulvasilenko/discordbot/discordbot/confify"
	"github.com/paulvasilenko/discordbot/discordbot/gamehighlighter"
	"github.com/paulvasilenko/discordbot/discordbot/homog"
	"github.com/paulvasilenko/discordbot/discordbot/quoter"
	"github.com/paulvasilenko/discordbot/discordbot/racing"
	"github.com/paulvasilenko/discordbot/discordbot/smileystats"
	"log"
	"log/syslog"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
)

var (
	token       = flag.String("token", "", "Bot Token")
	baseUrl     = flag.String("baseUrl", "", "Base url of local server where static is saved")
	faces       = flag.String("faces", "/home/sites/faces", "Faces to add to photo")
	basePath    = flag.String("basePath", "/var/www/static", "Path where files should be saved")
	MongoDbHost = flag.String("mongoDbHost", "127.0.0.1", "Mongo Db Host")
	MongoDbPort = flag.String("mongoDbPort", "27017", "Mongo Db Port")
	MySQLDbHost = flag.String("mysqlDbHost", "127.0.0.1", "Mysql Db Host")
	MySQLDbPort = flag.String("mysqlDbPort", "3306", "Mysql Db Port")
	MySQLDbUser = flag.String("mysqlDbUser", "artshadow", "Mysql Db Port")
	MySQLDbPass = flag.String("mysqlDbPass", "", "Mysql Db Port")
)

type Command interface {
	Subscribe(s *discordgo.Session)
	GetInfo() map[string]string
}

type Helper struct {
	CommandDocs map[string]string
}

func (h *Helper) AddDocs(s Command) {
	for k, v := range s.GetInfo() {
		h.CommandDocs[k] = v
	}
}

func (h *Helper) Subscribe(s *discordgo.Session) {
	s.AddHandler(h.MessageCreate)
}

func (h *Helper) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	if !strings.HasPrefix(m.Content, "!pandahelp") {
		return
	}

	args := strings.Split(m.Content, " ")

	if len(args) > 1 {
		info, ok := h.CommandDocs[args[1]]

		if !ok {
			return
		}

		s.ChannelMessageSend(m.ChannelID, info)

		return
	}

	message := ""

	for key, info := range h.CommandDocs {
		message += key + ": " + info + "\n\n"
	}

	s.ChannelMessageSend(m.ChannelID, message)
}

func main() {
	log.SetFlags(0)

	syslogWriter, err := syslog.New(syslog.LOG_INFO, "discordbot")

	if err == nil {
		log.SetOutput(syslogWriter)
	}

	flag.Parse()
	flagconfig.Parse()

	log.Printf("GOMAXPROCS is %d\n", runtime.GOMAXPROCS(0))

	if *token == "" {
		fmt.Println("No token provided")
	}

	dg, err := discordgo.New("Bot " + *token)

	if err != nil {
		fmt.Println(err)

		return
	}

	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)

	plugins := []Command{
		confify.NewConfify(*basePath, *baseUrl, *faces),
		quoter.NewQuoter(),
		homog.NewHomog()}

	for _, v := range plugins {
		v.Subscribe(dg)
	}

	gamehighlighterStruct, err := gamehighlighter.NewGameHighlighter(*MongoDbHost, *MongoDbPort)

	if err != nil {
		log.Println(err)
	} else {
		gamehighlighterStruct.Subscribe(dg)
	}

	plugins = append(plugins, gamehighlighterStruct)

	racingStruct, err := racing.NewRacing(
		*MongoDbHost, *MongoDbPort, *MySQLDbHost, *MySQLDbPort, *MySQLDbUser, *MySQLDbPass,
	)

	if err != nil {
		log.Println(err)
	} else {
		racingStruct.Subscribe(dg)
	}

	plugins = append(plugins, racingStruct)

	smileystatsStruct, err := smileystats.NewSmileyStats(*MySQLDbHost, *MySQLDbPort, *MySQLDbUser, *MySQLDbPass)

	if err != nil {
		log.Println(err)
	} else {
		smileystatsStruct.Subscribe(dg)
	}

	plugins = append(plugins, smileystatsStruct)

	helper := &Helper{CommandDocs: map[string]string{}}

	for _, plugin := range plugins {
		helper.AddDocs(plugin)
	}

	helper.Subscribe(dg)

	err = dg.Open()

	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}

	defer dg.Close()

	ctx, cancel := context.WithCancel(context.Background())

	go signalHandler(cancel)

	log.Println("PandaBot is now running.  Press CTRL-C to exit.")

	<-ctx.Done()
	return
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	_ = s.UpdateStatus(0, "Dirty Games")
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if strings.HasPrefix(m.Content, "!pandabot") {
		_, _ = s.ChannelMessageSend(m.ChannelID, "PandaBot")
		return
	}
}

func threeHundred(s *discordgo.Session, m *discordgo.MessageCreate) {
	if strings.Contains(m.Content, "300") || strings.Contains(m.Content, "триста") {
		s.ChannelMessageSend(m.ChannelID, "Отсоси у легалиста")
	}
}

func signalHandler(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	for {
		<-sigCh
		fmt.Println("Got stop signal, safely shutting down")
		cancel()
		return
	}
}

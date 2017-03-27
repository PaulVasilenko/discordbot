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
	"github.com/paulvasilenko/discordbot/discordbot/wiki"
	"log"
	"log/syslog"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
)

var (
	token    = flag.String("token", "", "Bot Token")
	baseUrl  = flag.String("baseUrl", "", "Base url of local server where static is saved")
	faces    = flag.String("faces", "/home/sites/faces", "Faces to add to photo")
	basePath = flag.String("basePath", "/var/www/static", "Path where files should be saved")
)

func main() {
	log.SetFlags(0)

	syslogWriter, err := syslog.New(syslog.LOG_INFO, "discordbot")

	if err == nil {
		log.SetOutput(syslogWriter)
	}

	flag.Parse()
	flagconfig.Parse()

	runtime.GOMAXPROCS(4)
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

	confifyStruct := confify.NewConfify(*basePath, *baseUrl, *faces)
	confifyStruct.Subscribe(dg)

	wikiStruct := wiki.NewWiki()
	wikiStruct.Subscribe(dg)

	homogStruct := homog.NewHomog()
	homogStruct.Subscribe(dg)

	gamehighlighterStruct, err := gamehighlighter.NewGameHighlighter()

	if err != nil {
		log.Println(err)
	} else {
		gamehighlighterStruct.Subscribe(dg)
	}

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

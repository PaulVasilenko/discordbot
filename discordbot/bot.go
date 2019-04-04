package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/facebookgo/flagconfig"
	"github.com/paulvasilenko/discordbot/discordbot/confify"
	"github.com/paulvasilenko/discordbot/discordbot/homog"
	"github.com/paulvasilenko/discordbot/discordbot/quoter"
	"github.com/paulvasilenko/discordbot/discordbot/racing"
	"github.com/paulvasilenko/discordbot/discordbot/smileystats"
	"log"
	"log/syslog"
	"net/http"
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
	serverPort  = flag.String("port", ":80", "Port which is reserved for http file server")
	MongoDbHost = flag.String("mongoDbHost", "127.0.0.1", "Mongo Db Host")
	MongoDbPort = flag.String("mongoDbPort", "27017", "Mongo Db Port")
	MySQLDbHost = flag.String("mysqlDbHost", "127.0.0.1", "Mysql Db Host")
	MySQLDbPort = flag.String("mysqlDbPort", "3306", "Mysql Db Port")
	MySQLDbUser = flag.String("mysqlDbUser", "artshadow", "Mysql Db Port")
	MySQLDbPass = flag.String("mysqlDbPass", "", "Mysql Db Port")
)

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
	fmt.Println("Opening connection with token: ", *token)

	dg, err := discordgo.New(*token)
	if err != nil {
		fmt.Println(err)

		return
	}

	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)

	c := confify.NewConfify(*basePath, *baseUrl, *faces)
	dg.AddHandler(c.MessageCreate)

	http.Handle("/", http.FileServer(http.Dir(*basePath)))
	go func() {
		if err := http.ListenAndServe(*serverPort, nil); err != nil {
			log.Println("failed to run fileserver: ", err)
		}
	}()

	q := quoter.NewQuoter()
	dg.AddHandler(q.MessageReactionAdd)

	h := homog.NewHomog()
	dg.AddHandler(h.MessageCreate)

	racingStruct, err := racing.NewRacing(
		*MongoDbHost, *MongoDbPort, *MySQLDbHost, *MySQLDbPort, *MySQLDbUser, *MySQLDbPass,
	)
	if err != nil {
		log.Println(err)
	} else {
		dg.AddHandler(racingStruct.MessageCreate)
	}



	smileystatsStruct, err := smileystats.NewSmileyStats(*MySQLDbHost, *MySQLDbPort, *MySQLDbUser, *MySQLDbPass)
	if err != nil {
		log.Println(err)
	} else {
		dg.AddHandler(smileystatsStruct.MessageCreate)
		dg.AddHandler(smileystatsStruct.MessageReactionAdd)
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
	err := s.UpdateStatus(0, "Dirty Games")
	if err != nil {
		fmt.Println("Error updating status:", err)
	}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if strings.HasPrefix(m.Content, "!pandabot") {
		_, err := s.ChannelMessageSend(m.ChannelID, "PandaBot")
		if err != nil {
			fmt.Println("ChannelMessageSend error:", err)
		}
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

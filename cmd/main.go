package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/paulvasilenko/discordbot/discordbot/sdr"

	"github.com/bwmarrin/discordgo"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/configor"
	"github.com/paulvasilenko/discordbot/discordbot/confify"
	"github.com/paulvasilenko/discordbot/discordbot/homog"
	"github.com/paulvasilenko/discordbot/discordbot/quoter"
	"github.com/paulvasilenko/discordbot/discordbot/racing"
	"github.com/paulvasilenko/discordbot/discordbot/smileystats"
	"github.com/paulvasilenko/discordbot/discordbot/tts"
	log "github.com/sirupsen/logrus"
	"gopkg.in/mgo.v2"
)

type Config struct {
	Token          string `required:"true" yaml:"Token"`
	BaseUrl        string `required:"true" yaml:"BaseUrl"`
	BasePath       string `required:"true" yaml:"BasePath"`
	FileServerPort string `default:":80" yaml:"FileServerPort"`
	FacesDir       string `required:"true" yaml:"FacesDir"`
	Mongo          struct {
		Host string `yaml:"Host"`
		Port string `yaml:"Port"`
	} `yaml:"Mongo"`
	Mysql struct {
		Host     string `yaml:"Host"`
		Port     string `yaml:"Port"`
		User     string `yaml:"User"`
		Password string `yaml:"Password"`
		Name     string `default:"pandabot" yaml:"Name"`
	} `yaml:"Mysql"`
	TTS struct {
		RequestURL string `yaml:"RequestURL"`
	} `yaml:"TTS"`
	SmileyStats struct {
		Blacklist []string `yaml:"Blacklist"`
	} `yaml:"SmileyStats"`
	SDR struct {
		Texts string `yaml:"Texts"`
		Gifts string `yaml:"Gifts"`
	} `yaml:"SDR"`
}

func main() {
	log.SetFormatter(&log.TextFormatter{})

	conf := Config{}
	err := configor.
		New(&configor.Config{
			ErrorOnUnmatchedKeys: true,
		}).
		Load(&conf, "config.yaml", "conf/config.yaml", "/etc/discordbot/config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log.Println("Opening connection with token: ", conf.Token)

	dg, err := discordgo.New(conf.Token)
	if err != nil {
		log.Fatalf("failed to open discord client: %v", err)
	}

	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)

	c := confify.NewConfify(conf.BasePath, conf.BaseUrl, conf.FacesDir)
	dg.AddHandler(c.MessageCreate)

	http.Handle("/", http.FileServer(http.Dir(conf.BasePath)))
	go func() {
		if err := http.ListenAndServe(conf.FileServerPort, nil); err != nil {
			log.Println("failed to run fileserver: ", err)
		}
	}()

	q := quoter.NewQuoter()
	dg.AddHandler(q.MessageReactionAdd)

	h := homog.NewHomog()
	dg.AddHandler(h.MessageCreate)

	textToSpeech := tts.NewTTS(&tts.TTSClient{
		Client:     &http.Client{},
		RequestURL: conf.TTS.RequestURL,
	})

	dg.AddHandler(textToSpeech.MessageReactionAdd)

	mysqlConn := initMysql(conf)
	mongodbConn := initMongo(conf)

	racingStruct, err := racing.NewRacing(mongodbConn, mysqlConn)
	if err != nil {
		log.Println(err)
	} else {
		dg.AddHandler(racingStruct.MessageCreate)
	}

	emotesStats := smileystats.NewSmileyStats(mysqlConn)
	dg.AddHandler(emotesStats.MessageCreate)
	dg.AddHandler(emotesStats.MessageReactionAdd)

	var (
		texts map[int]string
		gifts map[int]sdr.Gift
	)
	gr := errgroup.Group{}
	gr.Go(func() error {
		bytes, err := ioutil.ReadFile(conf.SDR.Texts)
		if err != nil {
			return err
		}
		return json.Unmarshal(bytes, &texts)
	})
	gr.Go(func() error {
		bytes, err := ioutil.ReadFile(conf.SDR.Gifts)
		if err != nil {
			return err
		}
		return json.Unmarshal(bytes, &gifts)
	})
	if err := gr.Wait(); err != nil {
		log.Fatal(err)
	}

	sdrPlugin, err := sdr.NewSDR(mongodbConn, texts, gifts)
	if err != nil {
		log.Println(err)
	} else {
		dg.AddHandler(sdrPlugin.MessageCreate)
	}

	if err = dg.Open(); err != nil {
		log.Fatalf("error opening Discord session: %v", err)
	}
	defer dg.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go signalHandler(cancel)

	log.Println("PandaBot is now running.  Press CTRL-C to exit.")
	<-ctx.Done()
	return
}

func initMysql(conf Config) *sql.DB {
	dsn := conf.Mysql.User + ":" +
		conf.Mysql.Password +
		"@tcp(" + conf.Mysql.Host + ":" + conf.Mysql.Port + ")/" +
		conf.Mysql.Name

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("failed to open mysql connection: %v", err)
	}

	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(50)
	db.SetConnMaxLifetime(1 * time.Second)

	return db
}

func initMongo(conf Config) *mgo.Session {
	session, err := mgo.Dial("mongodb://" + conf.Mongo.Host + ":" + conf.Mongo.Port)
	if err != nil {
		log.Fatalf("failed to init mongodb connection: %v", err)
	}
	return session
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

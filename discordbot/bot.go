package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/facebookgo/flagconfig"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	ImageRegex      string = `(https?:\/\/.*\.(?:png|jpg))`
	DownloadTimeout int    = 60
)

var (
	token    = flag.String("token", "", "Bot Token")
	baseUrl  = flag.String("baseUrl", "", "Base url of local server where static is saved")
	faces    = flag.String("faces", "/home/sites/faces", "Faces to add to photo")
	cascade  = flag.String("cascade", "/home/sites/faces/cascade.xml", "Haar cascade to find faces")
	basePath = "/var/www/static"
)

func main() {
	flag.Parse()
	flagconfig.Parse()

	runtime.GOMAXPROCS(4)
	fmt.Printf("GOMAXPROCS is %d\n", runtime.GOMAXPROCS(0))

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

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
	}

	defer dg.Close()

	ctx, cancel := context.WithCancel(context.Background())

	go signalHandler(cancel)

	fmt.Println("PandaBot is now running.  Press CTRL-C to exit.")

	<-ctx.Done()
	return
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	//  Set the playing status.
	_ = s.UpdateStatus(0, "Dirty Games")
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if strings.HasPrefix(m.Content, "!pandabot") {
		_, _ = s.ChannelMessageSend(m.ChannelID, "PandaBot")
		return
	}

	if strings.HasPrefix(m.Content, "!confify") {
		fmt.Println("Starting confify image")
		defer fmt.Println("Finishing confify image")

		imgCh := make(chan string)
		ticksWaiting := 1
		message, _ := s.ChannelMessageSend(m.ChannelID, "Processing"+strings.Repeat(".", ticksWaiting%4))

		regexpImage, err := regexp.Compile(ImageRegex)

		if err != nil {
			fmt.Println("Error: ", err)
			return
		}

		var imageString string

		if imageString = regexpImage.FindString(m.Content); imageString == "" {
			s.ChannelMessageEdit(m.ChannelID, message.ID, "Please, provide image link with PNG or JPEG extension")

			return
		}

		go processImage(imgCh, imageString)

		for {
			time.Sleep(200 * time.Millisecond)
			select {
			case image := <-imgCh:
				s.ChannelMessageEdit(m.ChannelID, message.ID, "Processed file: "+*baseUrl+image)

				if image == "" {
					s.ChannelMessageEdit(
						m.ChannelID,
						message.ID,
						"Error during processing, please, notify PandaSam about it")
				}

				return
			default:
				fmt.Println("Waiting for image being procesed")
				ticksWaiting += 1
				s.ChannelMessageEdit(m.ChannelID, message.ID, "Processing"+strings.Repeat(".", ticksWaiting%4))
				if ticksWaiting > 50 {
					s.ChannelMessageEdit(m.ChannelID, message.ID, "Processing time exceeed")
					return
				}
			}
		}

		return
	}
}

func processImage(imgCh chan<- string, imageString string) {
	fmt.Println("Started image processing")
	defer fmt.Println("Finished image processing")

	downloadedFilename, downloadedFilePath, err := downloadFromUrl(imageString, "", basePath)

	if err != nil {
		fmt.Println(err)
		imgCh <- ""
		return
	}

	outputFilePath := basePath + "/processed_" + downloadedFilename

	args := []string{
		"--faces", *faces,
		"--haar", *cascade,
		downloadedFilePath, " > ", outputFilePath}

	cmd := exec.Command("chrisify", args...)

	out, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Println("Non-zero exit code: " + err.Error() + ", " + string(out))
	}

	fmt.Println("Image processed, putting in channel")

	imgCh <- "processed_" + downloadedFilename
	return
}

func downloadFromUrl(dUrl string, filename string, path string) (string, string, error) {
	err := os.MkdirAll(path, 755)
	if err != nil {
		return "", "", errors.New(fmt.Sprintln("Error while creating folder", path, "-", err))
	}

	timeout := time.Duration(time.Duration(DownloadTimeout) * time.Second)
	client := &http.Client{
		Timeout: timeout,
	}
	request, err := http.NewRequest("GET", dUrl, nil)
	if err != nil {
		return "", "", errors.New(fmt.Sprintln("Error while downloading", dUrl, "-", err))
	}
	request.Header.Add("Accept-Encoding", "identity")
	response, err := client.Do(request)
	if err != nil {
		return "", "", errors.New(fmt.Sprintln("Error while downloading", dUrl, "-", err))
	}
	defer response.Body.Close()

	if filename == "" {
		filename = filenameFromUrl(response.Request.URL.String())
		for key, iHeader := range response.Header {
			if key == "Content-Disposition" {
				_, params, err := mime.ParseMediaType(iHeader[0])
				if err == nil {
					newFilename, err := url.QueryUnescape(params["filename"])
					if err != nil {
						newFilename = params["filename"]
					}
					if newFilename != "" {
						filename = newFilename
					}
				}
			}
		}
	}

	completePath := path + string(os.PathSeparator) + filename
	if _, err := os.Stat(completePath); err == nil {
		tmpPath := completePath
		i := 1
		for {
			completePath = tmpPath[0:len(tmpPath)-len(filepath.Ext(tmpPath))] +
				"-" + strconv.Itoa(i) + filepath.Ext(tmpPath)
			if _, err := os.Stat(completePath); os.IsNotExist(err) {
				break
			}
			i = i + 1
		}
		fmt.Printf("[%s] Saving possible duplicate (filenames match): %s to %s\n", time.Now().Format(time.Stamp), tmpPath, completePath)
	}

	bodyOfResp, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return "", "", errors.New(fmt.Sprintln("Could not read response", dUrl, "-", err))
	}

	contentType := http.DetectContentType(bodyOfResp)
	contentTypeParts := strings.Split(contentType, "/")

	if contentTypeParts[0] != "image" && contentTypeParts[0] != "video" {
		return "", "", errors.New(fmt.Sprintln("No image or video found at", dUrl))
	}

	err = ioutil.WriteFile(completePath, bodyOfResp, 0644)

	if err != nil {
		return "", "", errors.New(fmt.Sprintln("Error while writing to disk", dUrl, "-", err))
	}

	return filename, completePath, err
}

func filenameFromUrl(dUrl string) string {
	base := path.Base(dUrl)
	parts := strings.Split(base, "?")
	return parts[0]
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

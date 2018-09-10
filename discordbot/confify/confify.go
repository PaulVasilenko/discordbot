package confify

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"log"
	"regexp"
)

const (
	ImageRegex      string = `(?i)(https?:\/\/.*\.(?:png|jpe?g))`
	DownloadTimeout int    = 60
)

// Confify is struct which represents Confify plugin with it's configurations
type Confify struct {
	BasePath string
	BaseUrl  string
	Faces    string
}

func NewConfify(basePath, baseUrl, faces string) *Confify {
	return &Confify{BasePath: basePath, BaseUrl: baseUrl, Faces: faces}
}

// Subscribe is method which subscribes plugin to all needed events
func (c *Confify) Subscribe(s *discordgo.Session) {
	s.AddHandler(c.MessageCreate)
}

// GetInfo returns map of info message
func (c *Confify) GetInfo() map[string]string {
	return map[string]string{
		"!confify": `!confify [imageurl] - Replaces faces on image to faces from folder, uses Google Vision API.`,
	}
}

func (c *Confify) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	if !strings.HasPrefix(m.Content, "!confify") {
		return
	}

	log.Println("Starting confify image")
	defer log.Println("Finishing confify image")

	imgCh := make(chan string)
	ticksWaiting := 1
	message, _ := s.ChannelMessageSend(m.ChannelID, "Processing"+strings.Repeat(".", ticksWaiting%4))

	regexpImage, err := regexp.Compile(ImageRegex)

	if err != nil {
		log.Println("Error: ", err)
		return
	}

	var imageString string
	messages, err := s.ChannelMessages(m.ChannelID, 10, message.ID, "", "")
	if err != nil {
		log.Println(err)
		return
	}
	messages = append([]*discordgo.Message{m.Message}, messages...)

	loop:
	for _, m := range messages {
		if imageString = regexpImage.FindString(m.Content); imageString != "" {
			break loop
		}
		for _, a := range m.Attachments {
			if imageString = regexpImage.FindString(a.URL); imageString != "" {
				break loop
			}
		}
	}

	if imageString == "" {
		s.ChannelMessageEdit(m.ChannelID, message.ID, "Please, provide image link with PNG or JPEG extension")

		return
	}

	go c.processImage(imgCh, imageString)

	for {
		time.Sleep(500 * time.Millisecond)
		select {
		case image := <-imgCh:
			s.ChannelMessageEdit(m.ChannelID, message.ID, "Processed file: "+c.BaseUrl+image)

			if image == "" {
				s.ChannelMessageEdit(
					m.ChannelID,
					message.ID,
					"Error during processing, please, notify PandaSam about it")
			}

			return
		default:
			fmt.Println("Waiting for image being processed")
			ticksWaiting += 1
			s.ChannelMessageEdit(m.ChannelID, message.ID, "Processing"+strings.Repeat(".", ticksWaiting%4))
			if ticksWaiting > 50 {
				s.ChannelMessageEdit(m.ChannelID, message.ID, "Processing time exceeed")
				return
			}
		}
	}
}

func (c *Confify) processImage(imgCh chan<- string, imageString string) {
	fmt.Println("Started image processing")
	defer fmt.Println("Finished image processing")

	downloadedFilename, downloadedFilePath, err := downloadFromUrl(imageString, "", c.BasePath)

	if err != nil {
		fmt.Println(err)
		imgCh <- ""
		return
	}
	splittedString := strings.Split(downloadedFilename, ".")
	fileExtension := splittedString[len(splittedString)-1]
	outputFileName := fmt.Sprintf("%x", md5.Sum([]byte("processed_"+downloadedFilename+time.Now().Format(time.RFC3339Nano)))) + "." + fileExtension
	outputFilePath := c.BasePath + "/" + outputFileName

	args := []string{
		"--faces", c.Faces,
		downloadedFilePath}

	cmd := exec.Command("chrisify", args...)

	out, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Println("Non-zero exit code: " + err.Error() + ", " + string(out))
	}

	err = ioutil.WriteFile(outputFilePath, out, 0644)

	fmt.Println("Image processed, putting in channel")

	imgCh <- outputFileName
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

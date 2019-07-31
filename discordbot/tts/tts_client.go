package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

type TTSClient struct {
	*http.Client

	RequestURL string
}

type requestBody struct {
	Voice string `json:"voice"`
	Text  string `json:"text"`
}

type responseBody struct {
	URL string `json:"speak_url"`
	Err string `json:"error"`
}

func (c *TTSClient) Process(text string) (io.Reader, error) {
	var r io.Reader

	payload := requestBody{
		Voice: getVoice(lettersStats(text)),
		Text:  text,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode payload")
	}

	r = bytes.NewReader(b)

	req, err := http.NewRequest("POST", c.RequestURL, r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare request")
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to do request")
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	respBody := &responseBody{}

	if err := decoder.Decode(respBody); err != nil {
		return nil, err
	}

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("failed to process text, resp: %v", errors.New(respBody.Err))
	}

	resp, err = http.Get(respBody.URL)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func getVoice(stat LettersStat) string {
	if stat.Cyrillic/stat.Latin > 1 {
		return "Maxim"
	}

	return "Brian"
}

type LettersStat struct {
	Latin    float64
	Cyrillic float64
}

func lettersStats(text string) LettersStat {
	ret := LettersStat{}
	for _, r := range []rune(text) {
		if r <= 'я' && r >= 'А' {
			ret.Cyrillic += 1
		} else if r <= 'z' && r >= 'A' {
			ret.Latin += 1
		}
	}
	return ret
}

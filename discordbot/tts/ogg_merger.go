package tts

import (
	"bytes"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
)

const (
	//packetContinue = 0x1
)

func MergeOggStreams(processedParts []io.Reader) (io.Reader, error) {
	buf := bytes.NewBuffer([]byte{})
	for _, v := range processedParts {
		oggBytes, err := ioutil.ReadAll(v)
		if err != nil {
			return nil, errors.Wrap(err, "error reading bytes to buf")
		}

		// TODO: Modify segments to properly load segments

		//if i != 0 {
		//	ind := bytes.Index(oggBytes, []byte(`OggS`))
		//	oggBytes[ind+5] = packetContinue
		//}

		if _, err := buf.Write(oggBytes); err != nil {
			return nil, errors.Wrap(err, "error transferring data to buf")
		}
	}
	return buf, nil
}

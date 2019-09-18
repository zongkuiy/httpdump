package main

import (
	"bytes"
	"encoding/xml"
	"golang.org/x/net/html/charset"
	"io"
	"strings"
)

func xmlPrettify(src string) (string, error) {
	if len(strings.TrimSpace(src)) == 0 {
		return src, nil
	}

	decoder := xml.NewDecoder(bytes.NewBufferString(src))
	decoder.CharsetReader = charset.NewReaderLabel

	buf := new(bytes.Buffer)
	encoder := xml.NewEncoder(buf)
	encoder.Indent("", "    ")

	procInited := false

	for {
		t, err := decoder.Token()

		if err != nil && err != io.EOF {
			return src, err
		}

		if err == io.EOF {
			break
		}

		if t == nil {
			break
		}

		switch t.(type) {
		case xml.ProcInst:
			procInited = true
			err = encoder.EncodeToken(t)
		default:
			if procInited {
				err = encoder.EncodeToken(t)
			}
		}

		if err != nil {
			return "", err
		}
	}

	encoder.Flush()

	return buf.String(), nil
}

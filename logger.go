package main

import (
	"bytes"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	timestampFormat = "2006-01-02 15:04:05"
)

type LogFormatter struct{}

func (f *LogFormatter) Format(entry *log.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	_, err := fmt.Fprintf(b, "[notification-server] [%s] [%s] %s\n",
		entry.Time.Format(timestampFormat),
		strings.ToUpper(entry.Level.String()),
		entry.Message,
	)
	return b.Bytes(), err
}

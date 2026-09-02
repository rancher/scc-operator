package logging

import (
	"bytes"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type OptimizedSimpleFormatter struct{}

var _ logrus.Formatter = (*OptimizedSimpleFormatter)(nil)

func (o *OptimizedSimpleFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	if entry.Buffer != nil {
		buf := entry.Buffer
		o.writeLog(buf, entry)
		return buf.Bytes(), nil
	}

	// Paranoid but ultra-clean fallback: pre-allocate a local stack array
	var scratch [512]byte
	buf := bytes.NewBuffer(scratch[:0])
	o.writeLog(buf, entry)

	// Must copy here because the underlying array is on the local stack
	res := make([]byte, buf.Len())
	copy(res, buf.Bytes())
	return res, nil
}

func (o *OptimizedSimpleFormatter) writeLog(buf *bytes.Buffer, entry *logrus.Entry) {
	buf.WriteString(entry.Time.Format(time.DateTime))
	buf.WriteString(" [")
	buf.WriteString(strings.ToUpper(entry.Level.String()))
	buf.WriteString("] ")
	buf.WriteString(entry.Message)
	buf.WriteString("\n")
}

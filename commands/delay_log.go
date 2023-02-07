package commands

import (
	"github.com/mylxsw/asteria/level"
	"github.com/mylxsw/asteria/log"
)

type Logger struct {
	Events []string
}

func NewLogger() *Logger {
	return &Logger{Events: make([]string, 0)}
}

func (lo *Logger) Add(event string) {
	lo.Events = append(lo.Events, event)
}

func (lo *Logger) Flush() {
	log.GetDefaultConfig().LogWriter.Write(level.Info, "", "\n")
	for _, evt := range lo.Events {
		log.Infof(evt)
	}
}

package commands

import "github.com/mylxsw/asteria/log"

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
	for _, evt := range lo.Events {
		log.Infof(evt)
	}
}

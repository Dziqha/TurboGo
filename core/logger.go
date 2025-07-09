package core

import (
	"fmt"
	"log"
	"time"

	"github.com/fatih/color"
)

type LogLevel string
var DisableLogger bool

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
)

type Logger struct {
	Level LogLevel
}

var Log = &Logger{Level: DEBUG} // default

func (l *Logger) output(level LogLevel, msg string, args ...any) {
	if DisableLogger {
		return
	}
	if !l.shouldLog(level) {
		return
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formatted := fmt.Sprintf(msg, args...)

	colored := fmt.Sprintf("[%s] %s - %s", level, timestamp, formatted)
	switch level {
	case DEBUG:
		color.New(color.FgHiBlack).Println(colored)
	case INFO:
		color.New(color.FgGreen).Println(colored)
	case WARN:
		color.New(color.FgYellow).Println(colored)
	case ERROR:
		color.New(color.FgRed).Println(colored)
	default:
		log.Println(colored)
	}
}

func (l *Logger) shouldLog(level LogLevel) bool {
	order := map[LogLevel]int{
		DEBUG: 1,
		INFO:  2,
		WARN:  3,
		ERROR: 4,
	}
	return order[level] >= order[l.Level]
}

func (l *Logger) Debug(msg string, args ...any) { l.output(DEBUG, msg, args...) }
func (l *Logger) Info(msg string, args ...any)  { l.output(INFO, msg, args...) }
func (l *Logger) Warn(msg string, args ...any)  { l.output(WARN, msg, args...) }
func (l *Logger) Error(msg string, args ...any) { l.output(ERROR, msg, args...) }

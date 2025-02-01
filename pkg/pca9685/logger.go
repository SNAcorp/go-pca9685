package pca9685

import (
	"log"
)

type LogLevel int

const (
	// LogLevelBasic – базовый уровень логирования (выводятся только важные сообщения)
	LogLevelBasic LogLevel = iota
	// LogLevelDetailed – подробный уровень логирования (выводятся все сообщения)
	LogLevelDetailed
)

// Logger – минимальный интерфейс для логирования.
type Logger interface {
	Basic(msg string, args ...interface{})
	Detailed(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

type defaultLogger struct {
	level LogLevel
}

// NewDefaultLogger создаёт новый логгер с указанным уровнем.
func NewDefaultLogger(level LogLevel) Logger {
	return &defaultLogger{level: level}
}

func (l *defaultLogger) Basic(msg string, args ...interface{}) {
	log.Printf("[INFO] "+msg, args...)
}

func (l *defaultLogger) Detailed(msg string, args ...interface{}) {
	if l.level >= LogLevelDetailed {
		log.Printf("[DEBUG] "+msg, args...)
	}
}

func (l *defaultLogger) Error(msg string, args ...interface{}) {
	log.Printf("[ERROR] "+msg, args...)
}

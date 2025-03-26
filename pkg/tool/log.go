package tool

import (
	"log"
	"sync"
)

type Logger struct {
	debugMode bool
}

var (
	instance *Logger
	once     sync.Once
)

func NewLogger(debugMode bool) *Logger {
	m := debugMode == true
	return &Logger{debugMode: m}
}

func GetLogger(debugSetting bool) *Logger {
	// debugSetting := getDebugSetting()
	once.Do(func() {
		instance = NewLogger(debugSetting)
	})
	return instance
}

func (l *Logger) Println(v ...interface{}) {
	if l.debugMode {
		log.Println(append([]interface{}{"[DEBUG]"}, v...)...)
	}
}
func (l *Logger) Printf(format string, v ...interface{}) {
	if l.debugMode {
		log.Printf("[DEBUG] "+format, v...)
	}
}

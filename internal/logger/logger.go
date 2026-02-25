package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level defines the logging severity.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelError
	LevelNone
)

// Logger provides thread-safe structured logging for the engine.
// It is particularly useful for debugging UCI communication and search behavior
// in engine-vs-engine matches where stdout is reserved for the UCI protocol.
type Logger struct {
	mu      sync.Mutex
	level   Level
	writer  io.Writer
	enabled bool
}

var (
	instance *Logger
	once     sync.Once
)

// Get returns the singleton instance of the logger.
func Get() *Logger {
	once.Do(func() {
		instance = &Logger{
			level:   LevelDebug,
			writer:  os.Stderr,
			enabled: false,
		}
	})
	return instance
}

// SetLevel sets the minimum severity level to log.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetOutput sets the destination for log messages.
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writer = w
}

// SetFile opens a file for logging and enables the logger.
func (l *Logger) SetFile(path string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	l.SetOutput(f)
	l.SetEnabled(true)
	return nil
}

// SetEnabled toggles logging on or off globally.
func (l *Logger) SetEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = enabled
}

func (l *Logger) write(level Level, prefix string, format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.enabled || level < l.level || l.writer == nil {
		return
	}

	timestamp := time.Now().Format("15:04:05.000")
	msg := fmt.Sprintf(format, v...)
	fmt.Fprintf(l.writer, "[%s] [%s] %s\n", timestamp, prefix, msg)
}

// Debug logs a message at LevelDebug.
func (l *Logger) Debug(format string, v ...interface{}) {
	l.write(LevelDebug, "DEBUG", format, v...)
}

// Info logs a message at LevelInfo.
func (l *Logger) Info(format string, v ...interface{}) {
	l.write(LevelInfo, "INFO ", format, v...)
}

// Error logs a message at LevelError.
func (l *Logger) Error(format string, v ...interface{}) {
	l.write(LevelError, "ERROR", format, v...)
}

// Global convenience functions to use the singleton logger.

func Debug(format string, v ...interface{}) { Get().Debug(format, v...) }
func Info(format string, v ...interface{})  { Get().Info(format, v...) }
func Error(format string, v ...interface{}) { Get().Error(format, v...) }

func SetLevel(level Level)      { Get().SetLevel(level) }
func SetEnabled(enabled bool)   { Get().SetEnabled(enabled) }
func SetFile(path string) error { return Get().SetFile(path) }

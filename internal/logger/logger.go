package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

type (
	LogLevel string // LogLevel represents logging level
	Fields   map[string]any
)

// available log levels
const (
	DebugLogLevel LogLevel = "debug"
	InfoLogLevel  LogLevel = "info"
	WarnLogLevel  LogLevel = "warn"
	ErrorLogLevel LogLevel = "error"
	FatalLogLevel LogLevel = "fatal"
	PanicLogLevel LogLevel = "panic"
)

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level       LogLevel `json:"level"`
	Development bool     `json:"development"`

	// rolling log config
	LogFile    string `json:"log_file"`
	MaxSize    int    `json:"max_size"`
	MaxBackups int    `json:"max_backups"`
	MaxAge     int    `json:"max_age"`
	Compress   bool   `json:"compress"`
}

// Logger interface defines all logging methods
type Logger interface {
	Debug(msg string, fields ...any)
	Info(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Error(msg string, fields ...any)
	Fatal(msg string, fields ...any)
	Panic(msg string, fields ...any)
	WithFields(fields Fields) Logger
	WithError(err error) Logger
	WithContext(ctx context.Context) Logger
	Cleanup()
	ReleaseMap()
}

type logger struct {
	zl          zerolog.Logger
	fields      Fields
	mu          sync.RWMutex
	pool        *sync.Pool
	lumberjack  *lumberjack.Logger
	development bool
}

// use sync.Pool for map reuse
var mapPool = &sync.Pool{
	New: func() any {
		return make(Fields, 10)
	},
}

// NewLogger creates a new logger instance
func NewLogger(config *LoggerConfig) Logger {
	if config == nil {
		config = &LoggerConfig{
			Level:       DebugLogLevel,
			Development: true,
			LogFile:     "./logs/file.log",
			MaxSize:     100,
			MaxBackups:  3,
			MaxAge:      28,
			Compress:    true,
		}
	}

	var (
		output io.Writer
		lumber *lumberjack.Logger
	)
	if config.Development {
		// console output
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
	} else {
		// prod: rolling file output
		lumber = &lumberjack.Logger{
			Filename:   config.LogFile,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		}

		// Multi-writer for both file and stderr
		output = io.MultiWriter(lumber, os.Stderr)
	}

	zerolog.SetGlobalLevel(getZerologLevel(config.Level))

	zl := zerolog.New(output).
		With().
		Timestamp().
		CallerWithSkipFrameCount(4).
		// Int("pid", os.Getpid()).
		Logger()

	return &logger{
		zl:          zl,
		fields:      make(Fields),
		pool:        mapPool,
		lumberjack:  lumber,
		development: config.Development,
	}
}

// WithFields creates a new Logger instance with the provided fields added to the
// logger's fields. This allows additional context to be included in log entries.
func (l *logger) WithFields(fields Fields) Logger {
	// get new map from pool
	newFields := l.pool.Get().(Fields)

	// need lock: reading shared fields map
	func() {
		l.mu.RLock()
		defer l.mu.RUnlock()
		// copy existing fields
		for k, v := range l.fields {
			newFields[k] = v
		}
	}()

	// add new fields
	for k, v := range fields {
		newFields[k] = v
	}

	// create new logger instances to maintain immutability and prevent race conditions.
	// only the fields maps are unique to each logger instance.
	newLogger := &logger{
		zl:     l.zl,
		fields: newFields,
		pool:   l.pool,
	}

	// clean up and return old map to pool
	l.ReleaseMap()

	return newLogger
}

// WithError creates a new Logger instance with the provided error added to the
// logger's fields. This allows the error to be included in log entries.
func (l *logger) WithError(err error) Logger {
	newFields := l.pool.Get().(Fields)

	// need lock: reading shared fields map
	func() {
		l.mu.RLock()
		defer l.mu.RUnlock()
		// copy existing fields
		for k, v := range l.fields {
			newFields[k] = v
		}
	}()

	newFields["error"] = err.Error()

	// create new logger instances to maintain immutability and prevent race conditions.
	newLogger := &logger{
		zl:     l.zl,
		fields: newFields,
		pool:   l.pool,
	}

	// clean up and return old map to pool
	l.ReleaseMap()

	return newLogger
}

func (l *logger) WithContext(ctx context.Context) Logger {
	return nil
}

// logEvent is a helper method that logs an event with the logger's fields.
// It checks the provided fields for key-value pairs, and adds them to the event.
// It then adds the logger's fields to the event, and logs the message.
func (l *logger) logEvent(event *zerolog.Event, msg string, fields ...any) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// check if we have pairs of fields (key-value)
	if len(fields) > 0 {
		if len(fields)%2 != 0 { // odd number of fields means unpaired key-value
			event.Interface("UNPAIRED_FIELDS", fields)
		} else {
			// process key-value pairs
			for i := 0; i < len(fields); i += 2 {
				key, ok := fields[i].(string) // first item should be string key
				if !ok {
					// key isn't a string
					event.Interface("INVALID_KEY", fields[i])
					continue
				}
				event.Interface(key, fields[i+1]) // add key-value pair to log
			}
		}
	}

	for k, v := range l.fields {
		event.Interface(k, v)
	}

	event.Msg(msg)
}

// releaseMap is a helper method to release map back to pool
func (l *logger) ReleaseMap() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.fields != nil {
		clear(l.fields)
		l.pool.Put(l.fields)
		l.fields = nil // prevents accidental use of released map
	}
}

// Cleanup releases the logger's resources, including closing the lumberjack
// logger if it exists and is not in development mode.
func (l *logger) Cleanup() {
	l.ReleaseMap()

	// close lumberjack if in production
	if !l.development && l.lumberjack != nil {
		if err := l.lumberjack.Close(); err != nil {
			// log error using global stderr as logger might be unusable
			fmt.Fprintf(os.Stderr, "error closing lumberjack: %v\n", err)
		}
	}
}

// getZerologLevel converts a LogLevel to the corresponding zerolog.Level.
// If the provided LogLevel is not recognized, it defaults to zerolog.InfoLevel.
func getZerologLevel(level LogLevel) zerolog.Level {
	switch level {
	case DebugLogLevel:
		return zerolog.DebugLevel
	case InfoLogLevel:
		return zerolog.InfoLevel
	case WarnLogLevel:
		return zerolog.WarnLevel
	case ErrorLogLevel:
		return zerolog.ErrorLevel
	case FatalLogLevel:
		return zerolog.FatalLevel
	case PanicLogLevel:
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

func (l *logger) Debug(msg string, fields ...any) {
	l.logEvent(l.zl.Debug(), msg, fields...)
}

func (l *logger) Info(msg string, fields ...any) {
	l.logEvent(l.zl.Info(), msg, fields...)
}

func (l *logger) Warn(msg string, fields ...any) {
	l.logEvent(l.zl.Warn(), msg, fields...)
}

func (l *logger) Error(msg string, fields ...any) {
	l.logEvent(l.zl.Error(), msg, fields...)
}

func (l *logger) Fatal(msg string, fields ...any) {
	l.logEvent(l.zl.Fatal(), msg, fields...)
}

func (l *logger) Panic(msg string, fields ...any) {
	l.logEvent(l.zl.Panic(), msg, fields...)
}

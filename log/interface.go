// Copyright (c) 2018-2019 KIDTSUNAMI
// Author: alex@kidtsunami.com

package log

import (
	logpkg "log"
)

type LogFn func(...interface{})
type LogfFn func(string, ...interface{})

type Level int

const (
	LevelTrace Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
	LevelOff
	LevelInvalid
)

type Logger interface {
	Trace(v ...interface{})
	Tracef(f string, v ...interface{})
	Debug(v ...interface{})
	Debugf(f string, v ...interface{})
	Info(v ...interface{})
	Infof(f string, v ...interface{})
	Warn(v ...interface{})
	Warnf(f string, v ...interface{})
	Error(v ...interface{})
	Errorf(f string, v ...interface{})
	Fatal(v ...interface{})
	Fatalf(f string, v ...interface{})
	Level() Level
	SetLevel(Level) Logger
	Logger() *logpkg.Logger
	Clone(tag string) Logger
	// NewWriter(Level) io.Writer
	// Write(p []byte) (n int, err error)
}

// package level forwarders to the real logger implementation
func Trace(v ...interface{})            { Log.Trace(v...) }
func Tracef(s string, v ...interface{}) { Log.Tracef(s, v...) }
func Error(v ...interface{})            { Log.Error(v...) }
func Errorf(s string, v ...interface{}) { Log.Errorf(s, v...) }
func Warn(v ...interface{})             { Log.Warn(v...) }
func Warnf(s string, v ...interface{})  { Log.Warnf(s, v...) }
func Info(v ...interface{})             { Log.Info(v...) }
func Infof(s string, v ...interface{})  { Log.Infof(s, v...) }
func Debug(v ...interface{})            { Log.Debug(v...) }
func Debugf(s string, v ...interface{}) { Log.Debugf(s, v...) }
func Fatal(v ...interface{})            { Log.Fatal(v...) }
func Fatalf(s string, v ...interface{}) { Log.Fatalf(s, v...) }

// func Level() Level                      { return Log.Level() }
func SetLevel(l Level) Logger { Log.SetLevel(l); return Log }
func NewLogger(tag string) Logger {
	if b, ok := Log.(*Backend); ok {
		return b.NewLogger(tag)
	} else {
		return New(NewConfig()).NewLogger(tag)
	}
}

// LogClosure is a closure that can be printed with %v to be used to
// generate expensive-to-create data for a detailed log level and avoid doing
// the work if the data isn't printed.
type LogClosure func() string

// String invokes the log closure and returns the results string.
func (c LogClosure) String() string {
	return c()
}

// NewLogClosure returns a new closure over the passed function which allows
// it to be used as a parameter in a logging function that is only invoked when
// the logging level is such that the message will actually be logged.
func NewLogClosure(c func() string) LogClosure {
	return LogClosure(c)
}

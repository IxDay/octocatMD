package main

import (
	"fmt"
	"log"
	"os"
)

const (
	DEBUG int = iota + 1
	INFO
	WARN
	ERROR
	OFF
)

var (
	lvlMap = map[int]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
		OFF:   "",
	}
	std = log.New(os.Stderr, "", log.LstdFlags)
)

type (
	LoggerFunc func(int, string, ...interface{})
	Logger     interface {
		Log(level int, format string, a ...interface{})
	}
)

func (f LoggerFunc) Log(level int, format string, a ...interface{}) {
	f(level, format, a...)
}

func NewStdLogger(logger *log.Logger, level int) Logger {
	return LoggerFunc(func(l int, format string, a ...interface{}) {
		if l >= level {
			logger.Printf(fmt.Sprintf("[%s] ", lvlMap[l])+format, a...)
		}
	})
}

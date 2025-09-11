package log

import "github.com/sirupsen/logrus"

type Logger interface {
	Print(value ...interface{})
	Printf(format string, args ...interface{})
	Info(args ...interface{})
	Error(args ...interface{})
	Warn(args ...interface{})
	Fatalf(format string, args ...interface{})
}

var logger = logrus.New()

func GetLogger() Logger {
	return logger
}

func NewLog() Logger {
	return logger
}

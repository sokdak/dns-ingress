package cloudflare

import (
	"fmt"
	"github.com/go-logr/logr"
)

type CfDefaultLogger struct {
	Log logr.Logger
}

func NewCfDefaultLogger(log logr.Logger) *CfDefaultLogger {
	return &CfDefaultLogger{
		Log: log.WithName("cloudflare-client"),
	}
}

func (l *CfDefaultLogger) Printf(format string, v ...interface{}) {
	l.Log.Info(fmt.Sprintf(format, v...))
}

type CfNopLogger struct{}

func NewCfNopLogger() *CfNopLogger {
	return &CfNopLogger{}
}

func (l *CfNopLogger) Printf(format string, v ...interface{}) {}

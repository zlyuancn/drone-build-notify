package logger

import (
	"github.com/drone/drone-go/plugin/logger"
	"github.com/zlyuancn/zlog"
)

var Log = zlog.DefaultLogger

var _ logger.Logger = (*wrapLogger)(nil)

type wrapLogger struct {
	zlog.Logfer
}

func (w *wrapLogger) Debugln(args ...interface{}) {
	w.Debug(args...)
}

func (w *wrapLogger) Errorln(args ...interface{}) {
	w.Error(args...)
}

func (w *wrapLogger) Infoln(args ...interface{}) {
	w.Info(args...)
}

func (w *wrapLogger) Warnln(args ...interface{}) {
	w.Warn(args...)
}

func Init(debug bool, logPath string) {
	conf := zlog.DefaultConfig
	conf.Name = "drone-build-notify"
	conf.ShowInitInfo = false
	if debug {
		conf.Level = "debug"
	} else {
		conf.Level = "info"
	}
	if logPath != "" {
		conf.WriteToFile = true
		conf.Path = logPath
	}
	Log = zlog.New(conf)
}

func MakeLogger() logger.Logger {
	return &wrapLogger{Log}
}

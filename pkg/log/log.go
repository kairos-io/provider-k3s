package log

import (
	"os"

	"github.com/kairos-io/provider-k3s/pkg/version"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

func InitLogger(path string) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	logfile := &lumberjack.Logger{
		Filename:   f.Name(),
		MaxSize:    10,
		MaxBackups: 5,
		Compress:   true,
	}

	logrus.SetOutput(logfile)
	logrus.SetFormatter(K3sLogger{
		Version:   version.Version,
		Formatter: logrus.StandardLogger().Formatter,
	})
}

type K3sLogger struct {
	Version   string
	Formatter logrus.Formatter
}

func (l K3sLogger) Format(entry *logrus.Entry) ([]byte, error) {
	entry.Data["version"] = l.Version
	return l.Formatter.Format(entry)
}

package logger

import (
	"io"
	"log/slog"
	"os"
	"path"
	"sync"
	"time"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

type instance struct {
	logger *slog.Logger
	writer io.Closer
}

type loggerMap struct {
	lock      *sync.RWMutex
	instances map[string]instance
}

var (
	loggers = loggerMap{
		new(sync.RWMutex),
		make(map[string]instance),
	}
	config Config

	// LoggerByDay 按照天来划分的logger.
	LoggerByDay Logger
)

const (
	loggerByDayFormat = "2006-01-02.log"
)

func (l *loggerMap) Close(name string) error {
	l.lock.RLock()
	_, ok := l.instances[name]
	l.lock.RUnlock()

	if !ok {
		return nil
	}

	l.lock.Lock()
	defer l.lock.Unlock()
	i, ok := l.instances[name]
	if ok {
		if e := i.writer.Close(); e != nil {
			return e
		}
		delete(l.instances, name)
	}

	return nil
}

func (l *loggerMap) Get(name string) Logger {
	l.lock.RLock()
	i, ok := l.instances[name]
	l.lock.RUnlock()

	if !ok {
		var ws *slog.Logger
		var closer io.Closer
		logcfg := &slog.HandlerOptions{
			AddSource: config.AddSource,
			Level:     config.Loglevel.toLevel(),
		}
		if !config.StdOut {
			lumb := &lumberjack.Logger{
				Filename:  path.Join(config.Path, name),
				MaxSize:   config.MaxSize,
				LocalTime: true,

				MaxBackups: config.MaxBackups,
				MaxAge:     config.MaxAge,
				Compress:   config.Compress,
			}
			ws = slog.New(slog.NewJSONHandler(lumb, logcfg))
			closer = lumb
		} else {
			ws = slog.New(slog.NewJSONHandler(os.Stdout, logcfg))
			closer = io.NopCloser(os.Stdout)
		}
		i = instance{
			logger: ws,
			writer: closer,
		}

		l.lock.Lock()
		if tmp, ok := l.instances[name]; !ok {
			l.instances[name] = i
		} else {
			i = tmp
		}
		l.lock.Unlock()
	}

	return &GIDContext{
		l: i.logger,
	}
}

// ToEarlyMorningTimeDuration will 计算当前到第二日凌晨的时间.
func ToEarlyMorningTimeDuration(now time.Time) time.Duration {
	hour := 24 - now.Hour() - 1
	minute := 60 - now.Minute() - 1
	second := 60 - now.Second()

	return time.Duration(hour)*time.Hour +
		time.Duration(minute)*time.Minute +
		time.Duration(second)*time.Second
}

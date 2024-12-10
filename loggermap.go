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
	instances map[string]*instance
}

var (
	loggers = loggerMap{
		new(sync.RWMutex),
		make(map[string]*instance),
	}
	config Config

	// // LoggerByDay 按照天来划分的logger.
	// LoggerByDay Logger
)

// const (
// 	loggerByDayFormat = "2006-01-02.log"
// )

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

// Range 遍历 loggerMap 中的所有实例，并对每个实例应用给定的函数 f。
//
// 参数：
//
//	f: 一个接受两个参数的函数，第一个参数是实例的名称（string 类型），第二个参数是实例本身（slog.Logger）。
//	    如果函数 f 返回 false，则遍历将提前终止。
//
// 返回值：
//
//	无返回值。
func (l *loggerMap) Range(f func(name string, i *instance) bool) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	for k, v := range l.instances {
		if !f(k, v) {
			break
		}
	}

}

func (l *loggerMap) Get(name string) Logger {
	l.lock.RLock()
	i, ok := l.instances[name]
	l.lock.RUnlock()

	if !ok {
		i = newSlog(config, name)

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

func newSlog(cfg Config, name string) *instance {
	var ws *slog.Logger
	var closer io.Closer
	logcfg := &slog.HandlerOptions{
		AddSource: cfg.AddSource,
		Level:     cfg.Loglevel.toLevel(),
	}
	if !cfg.StdOut {
		lumb := &lumberjack.Logger{
			Filename:  path.Join(cfg.Path, name),
			MaxSize:   cfg.MaxSize,
			LocalTime: true,

			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
		ws = slog.New(slog.NewJSONHandler(lumb, logcfg))
		closer = lumb
	} else {
		ws = slog.New(slog.NewJSONHandler(os.Stdout, logcfg))
		closer = io.NopCloser(os.Stdout)
	}
	return &instance{
		logger: ws,
		writer: closer,
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

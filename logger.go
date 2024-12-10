package logger

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// Logger Logger.
type Logger interface {
	Debugw() func(msg string, kv ...any)
	Infow() func(msg string, kv ...any)
	Warnw() func(msg string, kv ...any)
	Errorw() func(msg string, kv ...any)
}

// Empty empty logger.
var (
	Empty        = &emptyLogger{}
	goroutineMap *sync.Map
)

type emptyLogger struct{}

// Debugw Debugw.
func (e *emptyLogger) Debugw() func(_ string, _ ...any) {
	return func(_ string, _ ...any) {}
}

// Infow Infow.
func (e *emptyLogger) Infow() func(_ string, _ ...any) {
	return func(_ string, _ ...any) {}
}

// Warnw Warnw.
func (e *emptyLogger) Warnw() func(_ string, _ ...any) {
	return func(_ string, _ ...any) {}
}

// Errorw Errorw.
func (e *emptyLogger) Errorw() func(_ string, _ ...any) {
	return func(_ string, _ ...any) {}
}

// Config logger config.
type Config struct {
	Loglevel LevelString
	StdOut   bool // 如果true，则 path、maxsize失效
	Path     string
	// MaxSize 单文件最大存储，单位MB
	MaxSize int

	// 最多备份数
	MaxBackups int
	// 备份最大保留天数
	MaxAge int
	// 是否压缩备份
	Compress bool

	// 是否添加调用函数信息
	AddSource bool
}

// InitLoggerWithConfig 使用config初始化logger.
func InitLoggerWithConfig(cfg Config, location *time.Location, gid *sync.Map) error {
	if !cfg.StdOut {
		if len(cfg.Path) == 0 {
			return errors.New("path empty")
		}
		if e := exists(cfg.Path); e != nil {
			return e
		} else if cfg.MaxSize <= 0 {
			return errors.New("MaxSize must be large than zero")
		}
	}
	config = cfg
	goroutineMap = gid

	// Fix time offset for Local
	// lt := time.FixedZone("Asia/Shanghai", 8*60*60)
	if location != nil {
		time.Local = location
	}

	// if !cfg.StdOut {
	// 	lastFile := time.Now().Format(loggerByDayFormat)
	// 	LoggerByDay = GetSugarLogger(lastFile)
	// 	go func() {
	// 		for {
	// 			now := time.Now()
	// 			if lastFile != now.Format(loggerByDayFormat) {
	// 				go func(name string) {
	// 					if e := loggers.Close(name); e != nil {
	// 						log.Println("writer.Close error", e.Error(), "File", name)
	// 					}
	// 				}(lastFile)

	// 				lastFile = now.Format(loggerByDayFormat)
	// 				LoggerByDay = GetSugarLogger(lastFile)
	// 			}
	// 			time.Sleep(ToEarlyMorningTimeDuration(now))
	// 		}
	// 	}()
	// }

	return nil
}

// InitLoggerWithLevel 使用String格式的level初始化logger.
// path 输出路径, 默认当前路径.
// logLevel 日志级别: debug,info,warn.
// location 日志文件名所属时区.
func InitLoggerWithLevel(path string, logLevel LevelString, location *time.Location, gid *sync.Map) error {
	return InitLoggerWithConfig(Config{
		Path:       path,
		Loglevel:   logLevel,
		MaxSize:    64,
		MaxBackups: 0,
		MaxAge:     0,
		Compress:   true,
		AddSource:  false,
		StdOut:     false,
	}, location, gid)
}

// ResetAllInstance 函数用于重置所有日志实例。
//
// 参数：
//
//	cfg: 配置对象，包含日志配置信息。
//
// 说明：
//
//	遍历所有的日志实例，关闭每个实例的写入器，并从实例映射中删除。
//	然后根据提供的配置对象重新创建新的日志实例，并添加到实例映射中。
//	如果在关闭写入器时发生错误，则记录错误信息。
func ResetAllInstance(cfg Config) {
	loggers.Range(func(name string, i *instance) bool {
		if e := i.writer.Close(); e != nil {
			log.Println("[logger] writer.Close error", e.Error(), "file", name)
		}
		*i = *(newSlog(cfg, name))
		return false
	})
}

// GetLogger to get logger.
func GetLogger(name string) Logger {
	return loggers.Get(name)
}

// GetSugarLogger to get SugaredLogger.
func GetSugarLogger(name string) Logger {
	return GetLogger(name)
}

// FlushAndCloseLogger flush and close logger.
func FlushAndCloseLogger(name string) error {
	return loggers.Close(name)
}

func exists(path string) error {
	stat, err := os.Stat(path)
	if err == nil {
		return nil
	} else if os.IsNotExist(err) {
		return errors.New("path is not exists: " + path)
	} else if stat != nil && !stat.IsDir() {
		return errors.New("path is not directory: " + path)
	} else if stat == nil {
		return errors.New("not directory: " + path)
	}

	return err
}

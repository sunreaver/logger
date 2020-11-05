package logger

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Shopify/sarama"
	"github.com/pkg/errors"
	"github.com/sunreaver/tomlanalysis/bytesize"
	"go.uber.org/zap"
)

// Logger Logger.
type Logger interface {
	Debugw(msg string, kv ...interface{})
	Infow(msg string, kv ...interface{})
	Warnw(msg string, kv ...interface{})
	Errorw(msg string, kv ...interface{})
	Panicw(msg string, kv ...interface{})
}

var (
	// Empty empty logger.
	Empty = &emptyLogger{}
	kl    KafkaLogger
)

type emptyLogger struct{}

// Debugw Debugw.
func (e *emptyLogger) Debugw(_ string, _ ...interface{}) {
}

// Infow Infow.
func (e *emptyLogger) Infow(_ string, _ ...interface{}) {
}

// Warnw Warnw.
func (e *emptyLogger) Warnw(_ string, _ ...interface{}) {
}

// Errorw Errorw.
func (e *emptyLogger) Errorw(_ string, _ ...interface{}) {
}

// Panicw Panicw.
func (e *emptyLogger) Panicw(msg string, _ ...interface{}) {
	panic(msg)
}

// Config logger config.
type LogConfig struct {
	Path     string      `toml:"path"`
	Loglevel LevelString `toml:"level"`
	// MaxSize 单文件最大存储，单位MB
	MaxSize bytesize.Int64 `toml:"max_size_one_file"`
	// 是否开启kafka
	EnableKafka bool `toml:"enable_kafka"`
	// kafka配置文件
	KafkaConfig KafkaConfig `toml:"kafka"`
}

type KafkaConfig struct {
	ClientID   string   `toml:"client_id"`
	RackID     string   `toml:"rack_id"`
	BufferSize int      `toml:"buf_size"`
	Address    []string `toml:"address"`
	Ack        int16    `toml:"ack"`
	Topic      string   `toml:"topic"`
}

// InitLoggerWithConfig 使用config初始化logger.
func InitLoggerWithConfig(cfg LogConfig, location *time.Location) error {
	if len(cfg.Path) == 0 {
		return errors.New("path empty")
	}
	if e := exists(cfg.Path); e != nil {
		return e
	} else if cfg.MaxSize <= 0 {
		return errors.New("MaxSize must be large than zero")
	}
	// init kafka
	if cfg.EnableKafka {
		if cfg.KafkaConfig.Topic == "" {
			return errors.New("Kafka Topic empty")
		} else if len(cfg.KafkaConfig.Address) == 0 {
			return errors.New("Kafka Address empty")
		} else if cfg.KafkaConfig.Ack > 1 || cfg.KafkaConfig.Ack < -1 {
			return errors.New("Unknown kafka Ack flag")
		}
		//
		if err := initKafka(&cfg); err != nil {
			return err
		}
	}

	config = cfg

	// Fix time offset for Local
	// lt := time.FixedZone("Asia/Shanghai", 8*60*60)
	if location != nil {
		time.Local = location
	}

	lastFile := time.Now().Format(loggerByDayFormat)
	LoggerByDay = GetSugarLogger(lastFile)
	go func() {
		for {
			now := time.Now()
			if lastFile != now.Format(loggerByDayFormat) {
				go func(name string) {
					if e := loggers.Close(name); e != nil {
						fmt.Println("writer.Close error", e.Error(), "File", name)
					}
				}(lastFile)

				lastFile = now.Format(loggerByDayFormat)
				LoggerByDay = GetSugarLogger(lastFile)
			}
			time.Sleep(ToEarlyMorningTimeDuration(now))
		}
	}()

	return nil
}

func initKafka(c *LogConfig) error {
	// 设置日志输入到Kafka的配置
	kf := sarama.NewConfig()
	kf.ClientID = c.KafkaConfig.ClientID
	if c.KafkaConfig.BufferSize > 0 {
		kf.ChannelBufferSize = c.KafkaConfig.BufferSize
	}
	kf.RackID = c.KafkaConfig.RackID

	// 等待服务器所有副本都保存成功后的响应
	kf.Producer.RequiredAcks = sarama.RequiredAcks(c.KafkaConfig.Ack)
	// 随机的分区类型
	kf.Producer.Partitioner = sarama.NewRandomPartitioner
	// 是否等待成功和失败后的响应,只有上面的RequireAcks设置不是NoReponse这里才有用.
	kf.Producer.Return.Successes = true
	kf.Producer.Return.Errors = true

	sarama.Logger = log.New(os.Stdout, "[sarama] ", log.LstdFlags)

	p, err := sarama.NewSyncProducer(c.KafkaConfig.Address, kf)
	if err != nil {
		return errors.New(fmt.Sprintf("connect kafka failed: %+v\n", err))
	}
	kl.Topic = c.KafkaConfig.Topic
	kl.Producer = p

	return nil
}

// InitLoggerWithLevel 使用String格式的level初始化logger.
// path 输出路径, 默认当前路径.
// logLevel 日志级别: debug,info,warn.
// location 日志文件名所属时区.
func InitLoggerWithLevel(path string, logLevel LevelString, location *time.Location) error {
	return InitLogger(path, logLevel.toLevel(), location)
}

// InitLogger 初始化.
// path 输出路径, 默认当前路径.
// logLevel 日志级别.
// location 日志文件名所属时区.
func InitLogger(path string, logLevel Level, location *time.Location) error {
	return InitLoggerWithConfig(LogConfig{
		Path:     path,
		Loglevel: logLevel.toLevelString(),
		MaxSize:  1024,
	}, location)
}

// GetLogger to get logger.
func GetLogger(name string) *zap.Logger {
	return loggers.Get(name)
}

// GetSugarLogger to get SugaredLogger.
func GetSugarLogger(name string) *zap.SugaredLogger {
	return GetLogger(name).Sugar()
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

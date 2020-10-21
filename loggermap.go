package logger

import (
	"fmt"
	"github.com/Shopify/sarama"
	"os"
	"path"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

type instance struct {
	logger *zap.Logger
	writer *lumberjack.Logger
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
	LoggerByDay *zap.SugaredLogger
)

const (
	loggerByDayFormat = "2006-01-02.log"
)

func localTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	t = t.Local()
	enc.AppendString(t.Format("2006-01-02 15:04:05"))
}

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
		if e := i.logger.Sync(); e != nil {
			return e
		}
		if e := i.writer.Close(); e != nil {
			return e
		}
		delete(l.instances, name)
	}

	return nil
}

func (l *loggerMap) Get(name string) *zap.Logger {
	l.lock.RLock()
	i, ok := l.instances[name]
	l.lock.RUnlock()

	if !ok {
		var allCore []zapcore.Core

		writer := &lumberjack.Logger{
			Filename: path.Join(config.Path, name),
			MaxSize:  config.MaxSize,
		}
		ws := zapcore.AddSync(writer)
		cfg := zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     localTimeEncoder,
			EncodeDuration: zapcore.NanosDurationEncoder,
		}
		//
		if config.EnableKafka {
			var (
				kl  KafkaLogger
				err error
			)
			// 设置日志输入到Kafka的配置
			kf := sarama.NewConfig()
			//等待服务器所有副本都保存成功后的响应
			kf.Producer.RequiredAcks = sarama.WaitForAll
			//随机的分区类型
			kf.Producer.Partitioner = sarama.NewRandomPartitioner
			//是否等待成功和失败后的响应,只有上面的RequireAcks设置不是NoReponse这里才有用.
			kf.Producer.Return.Successes = true
			kf.Producer.Return.Errors = true

			kl.Topic = config.KafkaConfig.Topic
			kl.Producer, err = sarama.NewSyncProducer(config.KafkaConfig.Address, kf)
			if err != nil {
				fmt.Printf("connect kafka failed: %+v\n", err)
				os.Exit(-1)
			}
			topicErrors := zapcore.AddSync(&kl)
			kafkaEncoder := zapcore.NewJSONEncoder(cfg)
			allCore = append(allCore, zapcore.NewCore(kafkaEncoder, topicErrors, config.Loglevel.toZapcoreLevel()))
		}
		allCore = append(allCore, zapcore.NewCore(zapcore.NewJSONEncoder(cfg), ws, config.Loglevel.toZapcoreLevel()))
		//
		logger := zap.New(
			zapcore.NewTee(allCore...),
		)
		i = instance{
			logger: logger,
			writer: writer,
		}

		l.lock.Lock()
		if tmp, ok := l.instances[name]; !ok {
			l.instances[name] = i
		} else {
			i = tmp
		}
		l.lock.Unlock()
	}

	return i.logger
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

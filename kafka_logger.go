package logger

import (
	"github.com/Shopify/sarama"
	"github.com/pkg/errors"
)

type KafkaConfig struct {
	ClientID   string   `toml:"client_id"`
	RackID     string   `toml:"rack_id"`
	BufferSize int      `toml:"buf_size"`
	Address    []string `toml:"address"`
	Ack        int16    `toml:"ack"`
	Topic      string   `toml:"topic"`
	Version    string   `toml:"version"`
}

type KafkaLogger struct {
	Producer sarama.AsyncProducer
	Topic    string
	done     chan struct{}
}

func (lk *KafkaLogger) Write(p []byte) (n int, err error) {
	msg := &sarama.ProducerMessage{}
	msg.Topic = lk.Topic
	msg.Value = sarama.ByteEncoder(p)
	select {
	case <-lk.done:
		lk.Producer.AsyncClose()
		return 0, errors.New("kafka done")
	case lk.Producer.Input() <- &sarama.ProducerMessage{
		Topic: lk.Topic,
		Value: sarama.ByteEncoder(p),
	}:
	}

	return len(p), nil
}

func (lk *KafkaLogger) Stop() {
	close(lk.done)
}

package logger

import (
	"fmt"

	"github.com/Shopify/sarama"
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
	Producer sarama.SyncProducer
	Topic    string
}

func (lk *KafkaLogger) Write(p []byte) (n int, err error) {
	msg := &sarama.ProducerMessage{}
	msg.Topic = lk.Topic
	msg.Value = sarama.ByteEncoder(p)
	_, _, err = lk.Producer.SendMessage(msg)
	if err != nil {
		fmt.Println(err.Error())

		return
	}

	return
}

package logger

import (
	"github.com/Shopify/sarama"
	json "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

type KafkaConfig struct {
	ClientID      string   `toml:"client_id"`
	RackID        string   `toml:"rack_id"`
	BufferSize    int      `toml:"buf_size"`
	Address       []string `toml:"address"`
	Topic         string   `toml:"topic"`
	Version       string   `toml:"version"`
	FilterMessage []string `toml:"filter_messages"`
	Ack           int16    `toml:"ack"`
}

// 此message是否输出到kafka
func (kc *KafkaConfig) Filter(msg string) bool {
	if kafkaFilter == nil {
		return true
	}
	kafkaFilterOnce.Do(func() {
		kafkaFilter = make(map[string]bool, len(kc.FilterMessage))
		for _, fm := range kc.FilterMessage {
			kafkaFilter[fm] = true
		}
		if len(kafkaFilter) == 0 {
			kafkaFilter = nil
		}
	})

	return kafkaFilter[msg]
}

type KafkaLogger struct {
	Producer sarama.AsyncProducer
	Topic    string
	done     chan struct{}
	cfg      KafkaConfig
}

type loggerFmt struct {
	Message string `json:"message"`
}

func (lk *KafkaLogger) Write(p []byte) (n int, err error) {
	var lf loggerFmt
	err = json.Unmarshal(p, &lf)
	if err != nil {
		return 0, errors.Wrap(err, "log format")
	}
	if !lk.cfg.Filter(lf.Message) {
		return len(p), nil
	}

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

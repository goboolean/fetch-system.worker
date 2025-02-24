package kafka

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Goboolean/common/pkg/resolver"
	"github.com/Goboolean/fetch-system.IaC/pkg/model"
	"github.com/Goboolean/fetch-system.worker/internal/util/otel"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

type Producer struct {
	producer *kafka.Producer

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func NewProducer(c *resolver.ConfigMap) (*Producer, error) {

	bootstrap_host, err := c.GetStringKey("BOOTSTRAP_HOST")
	if err != nil {
		return nil, err
	}

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":            bootstrap_host,
		"acks":                         -1,
		"go.delivery.reports":          true,
		"batch.num.messages":           10000,      // By default, a batch can buffer up to 10,000 messages to form a MessageSet for batch sending, enhancing performance.
		"batch.size":                   1000000,    //The limit for the total size of a MessageSet batch, by default not exceeding 1,000,000 bytes.
		"queue.buffering.max.ms":       5,          //The delay before transmitting message batches (MessageSets) to the Broker to buffer messages, 5 ms by default.
		"queue.buffering.max.messages": 100000,     //The total number of messages buffered by the Producer should not exceed 100,000.
		"queue.buffering.max.kbytes":   1048576,    //MessageSets for the Producer buffering messages.
		"message.send.max.retries":     2147483647, //Number of retries, 2,147,483,647 by default.
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	instance := &Producer{
		producer: p,
		wg:       sync.WaitGroup{},
		ctx:      ctx,
		cancel:   cancel,
	}

	instance.traceEvent(ctx, &instance.wg)
	return instance, nil
}

func (p *Producer) produceProtobufData(topic string, key string, value proto.Message) error {
	payload, err := proto.Marshal(value)
	if err != nil {
		return err
	}

	if err = p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic},
		Key:            []byte(key),
		Value:          payload,
	}, nil); err != nil {
		return err
	}
	return nil
}

func (p *Producer) ProduceProtobufTrade(productId string, platform string, market string, data *model.TradeProtobuf) error {
	topic := fmt.Sprintf("%s.%s.usa.t", platform, market)
	return p.produceProtobufData(topic, productId, data)
}

func (p *Producer) ProduceProtobufAggs(productId string, productType string, platform string, market string, data *model.AggregateProtobuf) error {
	topic := fmt.Sprintf("%s.%s.%s", platform, market, productType)
	return p.produceProtobufData(topic, productId, data)
}

func (p *Producer) produceJsonData(topic string, key int64, value interface{}) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	bs := make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(bs, key)

	if err = p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic},
		Key:            bs,
		Value:          payload,
	}, nil); err != nil {
		return err
	}
	return nil
}

func (p *Producer) ProduceJsonTrade(productId string, data *model.TradeJson) error {
	topic := fmt.Sprintf("stock.%s.t", productId)
	return p.produceJsonData(topic, data.Timestamp, data)
}

func (p *Producer) ProduceJsonAggs(productId string, productType string, data *model.AggregateJson) error {
	topic := fmt.Sprintf("%s.%s", productId, productType)
	return p.produceJsonData(topic, data.Timestamp, data)
}

func (p *Producer) Flush(ctx context.Context) (int, error) {

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(time.Hour)
	}

	left := p.producer.Flush(int(time.Until(deadline).Milliseconds()))
	if left != 0 {
		return left, ErrFailedToFlush
	}

	return 0, nil
}

func (p *Producer) traceEvent(ctx context.Context, wg *sync.WaitGroup) {

	go func() {
		wg.Add(1)
		defer wg.Done()

		for e := range p.producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.WithField("error", ev.TopicPartition.Error).
						Error("Failed to produce message")
					otel.KafkaProducerErrorCount.Add(ctx, 1)
				} else {
					log.WithField("topic", *ev.TopicPartition.Topic).
						WithField("partition", ev.TopicPartition.Partition).
						WithField("offset", ev.TopicPartition.Offset).
						Info("Produced message")
					otel.KafkaProducerSuccessCount.Add(ctx, 1)
				}
			case *kafka.Error:
				log.WithField("error", ev).
					Error("Failed to produce message")
				otel.KafkaProducerErrorCount.Add(ctx, 1)
			}
		}
	}()
}

func (p *Producer) Close() {
	p.producer.Close()
	p.cancel()
	p.wg.Wait()
}

func (p *Producer) Ping(ctx context.Context) error {
	// It requires ctx to be deadline set, otherwise it will return error
	// It will return error if there is no response within deadline
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(time.Hour)
	}

	remaining := time.Until(deadline)
	_, err := p.producer.GetMetadata(nil, true, int(remaining.Milliseconds()))
	return err
}

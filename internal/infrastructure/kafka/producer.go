package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Goboolean/common/pkg/resolver"
	"github.com/Goboolean/fetch-system.IaC/pkg/model"
	"github.com/Goboolean/fetch-system.worker/internal/util/otel"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"google.golang.org/protobuf/proto"
)

type Producer struct {
	producer *kafka.Producer

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// example:
//
//	p, err := NewProducer(&resolver.ConfigMap{
//	  "BOOTSTRAP_HOST": os.Getenv("KAFKA_BOOTSTRAP_HOST"),
//	})
func NewProducer(c *resolver.ConfigMap) (*Producer, error) {

	bootstrap_host, err := c.GetStringKey("BOOTSTRAP_HOST")
	if err != nil {
		return nil, err
	}

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":   bootstrap_host,
		"acks":                -1,
		"go.delivery.reports": true,
	})

	ctx, cancel := context.WithCancel(context.Background())

	instance := &Producer{
		producer: p,
		wg:       sync.WaitGroup{},
		ctx:      ctx,
		cancel:   cancel,
	}

	trace, err := c.GetStringKey("TRACER")
	if err != nil {
		return nil, err
	}

	switch trace {
	case "otel":
		instance.traceOtelEvent(ctx, &instance.wg)
		break
	case "none":
		instance.traceEvent(ctx, &instance.wg)
		break
	default:
		return nil, fmt.Errorf("invalid trace option: %s", trace)
	}
	return instance, nil
}

func (p *Producer) produce(topic string, msg proto.Message) error {
	payload, err := proto.Marshal(msg)

	if err = p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic},
		Value:          payload,
	}, nil); err != nil {
		return err
	}

	return nil
}

func (p *Producer) ProduceTrade(productId string, data *model.Trade) error {
	topic := fmt.Sprintf("%s.%s", productId, "t")
	return p.produce(topic, data)
}

func (p *Producer) ProduceAggs(productId string, productType string, data *model.Aggregate) error {
	topic := fmt.Sprintf("%s.%s", productId, productType)
	return p.produce(topic, data)
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

		for range p.producer.Events() {}
	}()
}

func (p *Producer) traceOtelEvent(ctx context.Context, wg *sync.WaitGroup) {

	go func() {
		wg.Add(1)
		defer wg.Done()

		for e := range p.producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					otel.KafkaProducerErrorCount.Add(ctx, 1)
				} else {
					otel.KafkaProducerSuccessCount.Add(ctx, 1)
				}
			case *kafka.Error:
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

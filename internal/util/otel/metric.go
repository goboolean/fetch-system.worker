package otel

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

var (
	KafkaProducerSuccessCount metric.Int64Counter
	KafkaProducerErrorCount   metric.Int64Counter
	ETCDErrorCount            metric.Int64Counter
	MockGeneratorCount        metric.Int64Counter
	PolygonStockErrorCount    metric.Int64Counter
	PolygonStockReceivedCount metric.Int64Counter
	KISStockErrorCount        metric.Int64Counter
	KISStockReceivedCount     metric.Int64Counter
	ProductSubscribedCount    metric.Int64Counter
	ProductReceivedCount      metric.Int64Counter
)

func initMetric(meter metric.Meter) (err error) {
	KafkaProducerSuccessCount, err = meter.Int64Counter("kafka.producer.success.count")
	if err != nil {
		return
	}
	KafkaProducerSuccessCount.Add(context.Background(), 0)

	KafkaProducerErrorCount, err = meter.Int64Counter("kafka.producer.error.count")
	if err != nil {
		return
	}
	KafkaProducerErrorCount.Add(context.Background(), 0)

	ETCDErrorCount, err = meter.Int64Counter("etcd.error.count")
	if err != nil {
		return
	}
	ETCDErrorCount.Add(context.Background(), 0)

	MockGeneratorCount, err = meter.Int64Counter("mock.generator.count")
	if err != nil {
		return
	}
	MockGeneratorCount.Add(context.Background(), 0)

	PolygonStockErrorCount, err = meter.Int64Counter("polygon.stock.error.count")
	if err != nil {
		return
	}
	PolygonStockErrorCount.Add(context.Background(), 0)

	PolygonStockReceivedCount, err = meter.Int64Counter("polygon.stock.received.count")
	if err != nil {
		return
	}
	PolygonStockReceivedCount.Add(context.Background(), 0)

	KISStockErrorCount, err = meter.Int64Counter("kis.stock.error.count")
	if err != nil {
		return
	}
	KISStockErrorCount.Add(context.Background(), 0)

	KISStockReceivedCount, err = meter.Int64Counter("kis.stock.received.count")
	if err != nil {
		return
	}
	KISStockReceivedCount.Add(context.Background(), 0)

	ProductSubscribedCount, err = meter.Int64Counter("product.subscribed.count")
	if err != nil {
		return
	}
	ProductSubscribedCount.Add(context.Background(), 0)

	ProductReceivedCount, err = meter.Int64Counter("product.received.count")
	if err != nil {
		return
	}
	ProductReceivedCount.Add(context.Background(), 0)

	return
}

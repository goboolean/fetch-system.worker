// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package wire

import (
	"context"
	"fmt"
	"github.com/Goboolean/common/pkg/resolver"
	"github.com/Goboolean/fetch-system.worker/internal/adapter"
	"github.com/Goboolean/fetch-system.worker/internal/domain/port/out"
	"github.com/Goboolean/fetch-system.worker/internal/domain/service/pipe"
	"github.com/Goboolean/fetch-system.worker/internal/domain/service/task"
	"github.com/Goboolean/fetch-system.worker/internal/domain/vo"
	"github.com/Goboolean/fetch-system.worker/internal/infrastructure/etcd"
	"github.com/Goboolean/fetch-system.worker/internal/infrastructure/kafka"
	"github.com/Goboolean/fetch-system.worker/internal/infrastructure/kis"
	"github.com/Goboolean/fetch-system.worker/internal/infrastructure/mock"
	"github.com/Goboolean/fetch-system.worker/internal/infrastructure/polygon"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
)

// Injectors from wire_setup.go:

func InitializeKafkaProducer(ctx context.Context) (out.DataDispatcher, func(), error) {
	configMap := ProvideKafkaConfig()
	producer, cleanup, err := ProvideKafkaProducer(ctx, configMap)
	if err != nil {
		return nil, nil, err
	}
	worker := ProvideWorkerConfig()
	dataDispatcher := adapter.NewKafkaAdapter(producer, worker)
	return dataDispatcher, func() {
		cleanup()
	}, nil
}

func InitializeETCDClient(ctx context.Context) (out.StorageHandler, func(), error) {
	configMap := ProvideETCDConfig()
	client, cleanup, err := ProvideETCDClient(ctx, configMap)
	if err != nil {
		return nil, nil, err
	}
	storageHandler := adapter.NewETCDAdapter(client)
	return storageHandler, func() {
		cleanup()
	}, nil
}

func InitializePolygonStockClient(ctx context.Context) (out.DataFetcher, func(), error) {
	configMap := ProvidePolygonConfig()
	stocksClient, cleanup, err := ProvidePolygonStockClient(ctx, configMap)
	if err != nil {
		return nil, nil, err
	}
	dataFetcher, err := adapter.NewStockPolygonAdapter(stocksClient)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	return dataFetcher, func() {
		cleanup()
	}, nil
}

func InitializePolygonOptionClient(ctx context.Context) (out.DataFetcher, func(), error) {
	configMap := ProvidePolygonConfig()
	optionClient, cleanup, err := ProvidePolygonOptionClient(ctx, configMap)
	if err != nil {
		return nil, nil, err
	}
	dataFetcher, err := adapter.NewOptionPolygonAdapter(optionClient)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	return dataFetcher, func() {
		cleanup()
	}, nil
}

func InitializePolygonCryptoClient(ctx context.Context) (out.DataFetcher, func(), error) {
	configMap := ProvidePolygonConfig()
	cryptoClient, cleanup, err := ProvidePolygonCryptoClient(ctx, configMap)
	if err != nil {
		return nil, nil, err
	}
	dataFetcher, err := adapter.NewCryptoPolygonAdapter(cryptoClient)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	return dataFetcher, func() {
		cleanup()
	}, nil
}

func InitializeKISStockClient(ctx context.Context) (out.DataFetcher, func(), error) {
	configMap := ProvideKISConfig()
	client, err := kis.New(configMap)
	if err != nil {
		return nil, nil, err
	}
	dataFetcher, err := adapter.NewStockKISAdapter(client)
	if err != nil {
		return nil, nil, err
	}
	return dataFetcher, func() {
	}, nil
}

func InitializeMockGenerator(ctx context.Context) (out.DataFetcher, func(), error) {
	configMap := ProvideMockGeneratorConfig()
	client, cleanup, err := ProvideMockGenerator(configMap)
	if err != nil {
		return nil, nil, err
	}
	dataFetcher, err := adapter.NewMockGeneratorAdapter(client)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	return dataFetcher, func() {
		cleanup()
	}, nil
}

func InitializeETCD() (*etcd.Client, error) {
	configMap := ProvideETCDConfig()
	client, err := etcd.New(configMap)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func InitializePipeManager(dataDispatcher out.DataDispatcher, dataFetcher out.DataFetcher) (*pipe.Manager, error) {
	manager := pipe.New(dataFetcher, dataDispatcher)
	return manager, nil
}

func InitializeTaskManager(handler pipe.Handler, storageHandler out.StorageHandler) (*task.Manager, error) {
	worker := ProvideWorkerConfig()
	manager, err := task.New(worker, storageHandler, handler)
	if err != nil {
		return nil, err
	}
	return manager, nil
}

// wire_setup.go:

func ProvideOtelConfig() *resolver.ConfigMap {
	return &resolver.ConfigMap{
		"OTEL_ENDPOINT": os.Getenv("OTEL_ENDPOINT"),
	}
}

func ProvideKafkaConfig() *resolver.ConfigMap {
	return &resolver.ConfigMap{
		"BOOTSTRAP_HOST": os.Getenv("KAFKA_BOOTSTRAP_HOST"),
		"TRACER":         "otel",
	}
}

func ProvideMockGeneratorConfig() *resolver.ConfigMap {
	return &resolver.ConfigMap{
		"MODE":               "BASIC",
		"STANDARD_DEVIATION": 100,
		"PRODUCT_COUNT":      10,
		"PRODUCTION_RATE":    10,
		"BUFFER_SIZE":        100000,
	}
}

func ProvideKISConfig() *resolver.ConfigMap {
	return &resolver.ConfigMap{
		"APPKEY":      os.Getenv("KIS_APPKEY"),
		"SECRET":      os.Getenv("KIS_SECRET"),
		"MODE":        os.Getenv("MODE"),
		"BUFFER_SIZE": 100000,
	}
}

func ProvidePolygonConfig() *resolver.ConfigMap {
	return &resolver.ConfigMap{
		"SECRET_KEY":  os.Getenv("POLYGON_SECRET_KEY"),
		"BUFFER_SIZE": 100000,
	}
}

func ProvideETCDConfig() *resolver.ConfigMap {
	return &resolver.ConfigMap{
		"HOST": os.Getenv("ETCD_HOST"),
	}
}

func ProvideWorkerConfig() *vo.Worker {
	return &vo.Worker{
		ID:       uuid.New().String(),
		Platform: vo.Platform(os.Getenv("PLATFORM")),
		Market:   vo.Market(os.Getenv("MARKET")),
	}
}

func ProvideKafkaProducer(ctx context.Context, c *resolver.ConfigMap) (*kafka.Producer, func(), error) {
	producer, err := kafka.NewProducer(c)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to create kafka producer")
	}
	if err := producer.Ping(ctx); err != nil {
		return nil, nil, errors.Wrap(err, "Failed to send ping to kafka producer")
	}
	logrus.Info("Kafka producer is ready")

	return producer, func() {
		producer.Close()
		logrus.Info("Kafka producer is successfully closed")
	}, nil
}

func ProvideETCDClient(ctx context.Context, c *resolver.ConfigMap) (*etcd.Client, func(), error) {
	client, err := etcd.New(c)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to create etcd client")
	}
	if err := client.Ping(ctx); err != nil {
		return nil, nil, errors.Wrap(err, "Failed to send ping to etcd client")
	}
	logrus.Info("ETCD client is ready")

	return client, func() {
		client.Close()
		logrus.Info("ETCD client is successfully closed")
	}, nil
}

func ProvidePolygonStockClient(ctx context.Context, c *resolver.ConfigMap) (*polygon.StocksClient, func(), error) {
	client, err := polygon.NewStocksClient(c)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to create polygon client")
	}
	if err := client.Ping(ctx); err != nil {
		return nil, nil, errors.Wrap(err, "Failed to send ping to polygon client")
	}
	logrus.Info("Polygon client is ready")

	return client, func() {
		client.Close()
		logrus.Info("Polygon client is successfully closed")
	}, nil
}

func ProvidePolygonOptionClient(ctx context.Context, c *resolver.ConfigMap) (*polygon.OptionClient, func(), error) {
	client, err := polygon.NewOptionClient(c)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to create polygon client")
	}
	if err := client.Ping(ctx); err != nil {
		return nil, nil, errors.Wrap(err, "Failed to send ping to polygon client")
	}
	logrus.Info("Polygon option client is ready")

	return client, func() {
		client.Close()
		logrus.Info("Polygon client is successfully closed")
	}, nil
}

func ProvidePolygonCryptoClient(ctx context.Context, c *resolver.ConfigMap) (*polygon.CryptoClient, func(), error) {
	client, err := polygon.NewCryptoClient(c)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to create polygon client")
	}
	if err := client.Ping(ctx); err != nil {
		return nil, nil, errors.Wrap(err, "Failed to send ping to polygon client")
	}
	logrus.Info("Polygon client is ready")

	return client, func() {
		client.Close()
	}, nil
}

func ProvideKISStockClient(ctx context.Context, c *resolver.ConfigMap) (*kis.Client, func(), error) {
	client, err := kis.New(c)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to create kis client")
	}
	if err := client.Ping(ctx); err != nil {
		return nil, nil, errors.Wrap(err, "Failed to send ping to kis client")
	}
	logrus.Info("KIS client is ready")

	return client, func() {
		client.Close()
		logrus.Info("KIS client is successfully closed")
	}, nil
}

func ProvideMockGenerator(c *resolver.ConfigMap) (*mock.Client, func(), error) {
	client, err := mock.New(c)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to create mock client")
	}
	logrus.Info("Mock client is ready")

	return client, func() {
		client.Close()
		logrus.Info("Mock client is successfully closed")
	}, nil
}

func InitializeFetcher(ctx context.Context) (out.DataFetcher, func(), error) {
	switch os.Getenv("PLATFORM") {
	case "POLYGON":
		switch os.Getenv("MARKET") {
		case "STOCK":
			return InitializePolygonStockClient(ctx)
		case "OPTION":
			return InitializePolygonOptionClient(ctx)
		case "CRYPTO":
			return InitializePolygonCryptoClient(ctx)
		default:
			return nil, nil, fmt.Errorf("invalid market: %s", os.Getenv("MARKET"))
		}
	case "KIS":
		switch os.Getenv("MARKET") {
		case "STOCK":
			return InitializeKISStockClient(ctx)
		default:
			return nil, nil, fmt.Errorf("invalid market: %s", os.Getenv("MARKET"))
		}
	case "MOCK":
		return InitializeMockGenerator(ctx)
	default:
		return nil, nil, fmt.Errorf("invalid platform")
	}
}

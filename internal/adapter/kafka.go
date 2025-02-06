package adapter

import (
	"github.com/Goboolean/fetch-system.IaC/pkg/model"
	"github.com/Goboolean/fetch-system.worker/internal/domain/port/out"
	"github.com/Goboolean/fetch-system.worker/internal/domain/vo"
	"github.com/Goboolean/fetch-system.worker/internal/infrastructure/kafka"
	"github.com/sirupsen/logrus"
)

type KafkaAdapter struct {
	p *kafka.Producer
	w *vo.Worker
}

func NewKafkaAdapter(p *kafka.Producer, w *vo.Worker) out.DataDispatcher {
	return &KafkaAdapter{
		p: p,
		w: w,
	}
}

func (a *KafkaAdapter) OutputStream(ch <-chan *vo.Trade) error {
	go func() {
		for trade := range ch {
			data := &model.TradeJson{
				Price:     trade.Price,
				Size:      trade.Size,
				Timestamp: trade.Timestamp.Unix(),
			}
			logrus.WithFields(logrus.Fields{
				"id":     trade.ID,
				"symbol": trade.Symbol,
				"data":   data,
			}).Info("Producing trade")
			if err := a.p.ProduceJsonTrade(trade.Symbol, data); err != nil {
				logrus.WithFields(logrus.Fields{
					"error": err,
					"id":    trade.ID,
					"data":  data,
				}).Panic("Failed to produce trade")
			}
			/*
				if err := a.p.ProduceProtobufTrade(trade.ID, string(a.w.Platform), string(a.w.Market), data); err != nil {
					logrus.WithFields(logrus.Fields{
						"error": err,
						"id":    trade.ID,
						"data":  data,
					}).Panic("Failed to produce trade")
				}
			*/
		}
	}()
	return nil
}

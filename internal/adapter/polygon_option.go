package adapter

import (
	"context"
	"time"

	"github.com/Goboolean/fetch-system.worker/internal/domain/port/out"
	"github.com/Goboolean/fetch-system.worker/internal/domain/vo"
	"github.com/Goboolean/fetch-system.worker/internal/infrastructure/polygon"
)

type OptionPolygonAdapter struct {
	c *polygon.OptionClient
}

func NewOptionPolygonAdapter(c *polygon.OptionClient) (out.DataFetcher, error) {
	return &OptionPolygonAdapter{
		c: c,
	}, nil
}

func (a *OptionPolygonAdapter) InputStream(ctx context.Context, symbol ...string) (<-chan *vo.Trade, error) {

	ch := make(chan *vo.Trade)

	polyonCh, err := a.c.Subscribe()
	if err != nil {
		return nil, err
	}

	go func() {
		for v := range polyonCh {
			ch <- &vo.Trade{
				Symbol: v.Symbol,
				TradeDetail: vo.TradeDetail{
					Price:     v.Price,
					Size:      v.Size,
					Timestamp: time.Unix(v.Timestamp/1000, v.Timestamp%1000*1000000),
				},
			}
		}
	}()

	return ch, nil
}

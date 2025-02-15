package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Goboolean/fetch-system.worker/internal/domain/port/out"
	"github.com/Goboolean/fetch-system.worker/internal/domain/service/pipe"
	"github.com/Goboolean/fetch-system.worker/internal/domain/vo"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var ConfigRetry = 3
var ConfigTimeout = 3 * time.Second

type Manager struct {
	s out.StorageHandler
	p pipe.Handler

	worker    *vo.Worker
	primaryID string

	connCh chan struct{}
	promCh chan struct{}
	ttlCh  chan struct{}

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New(worker *vo.Worker, s out.StorageHandler, p pipe.Handler) (*Manager, error) {
	if worker.ID == "" {
		return nil, fmt.Errorf("worker id is empty")
	}

	if worker.Platform == "" {
		return nil, fmt.Errorf("worker platform is empty")
	}

	if worker.Market == "" {
		return nil, fmt.Errorf("worker market is empty")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		s:      s,
		p:      p,
		worker: worker,

		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (m *Manager) hasPrimary(workers []vo.Worker) (bool, string) {
	for _, worker := range workers {
		if worker.Status == vo.WorkerStatusPrimary {
			return true, worker.ID
		}
	}
	return false, ""
}

func (m *Manager) RegisterWorker(ctx context.Context) error {
	mu, err := m.s.Mutex(ctx, out.MutexKeyWorker)
	if err != nil {
		return errors.Wrap(err, "Failed to create mutex")
	}

	if err := mu.Lock(ctx); err != nil {
		return errors.Wrap(err, "Failed to aquire lock")
	}
	defer mu.Unlock(ctx)

	workers, err := m.s.GetAllWorker(ctx)
	if err != nil {
		return errors.Wrap(err, "Failed to query all workers from storage")
	}

	var isSecondary bool
	isSecondary, m.primaryID = m.hasPrimary(workers)

	if isSecondary {
		m.worker.Status = vo.WorkerStatusSecondary
	} else {
		m.worker.Status = vo.WorkerStatusPrimary
	}

	log.WithFields(log.Fields{
		"existingWorkers": len(workers),
		"status":          m.worker.Status,
	}).Info("Worker info aquired")

	connCh, err := m.s.CreateConnection(ctx, m.worker.ID)
	if err != nil {
		return errors.Wrap(err, "Failed to create connection")
	}
	m.connCh = connCh
	log.Info("Connection is successfully established")

	if err := m.s.RegisterWorker(ctx, *m.worker); err != nil {
		return errors.Wrap(err, "Failed to register worker")
	}
	log.WithFields(log.Fields{
		"workerId": m.worker.ID,
		"platform": m.worker.Platform,
		"market":   m.worker.Market,
		"status":   m.worker.Status,
	}).Info("Worker is successfully registered")

	products, err := m.s.GetProducts(ctx, m.worker.Platform, m.worker.Market)
	if err != nil {
		return errors.Wrap(err, "Failed to get products")
	}

	pipeErr := m.p.RunStreamingPipe(ctx, products)
	/*
		if !isSecondary {
			pipeErr = m.p.RunStreamingPipe(ctx, products)
			log.WithField("products", len(products)).Info("Streaming pipe is started")
		} else {
			pipeErr = m.p.RunStoringPipe(ctx, products)
			log.WithField("products", len(products)).Info("Storing pipe is started")
		}
	*/

	if pipeErr != nil {
		if err := m.s.UpdateWorkerStatus(context.Background(), m.worker.ID, vo.WorkerStatusExitedRegisterFailed); err != nil {
			return errors.Wrap(err, pipeErr.Error())
		}
		return errors.Wrap(pipeErr, "Failed to run pipe")
	}
	log.WithField("products", len(products)).Info("Streaming pipe is started")

	if isSecondary {
		go m.trackChan()

		m.promCh, err = m.s.WatchPromotion(ctx, m.primaryID)
		if err != nil {
			return errors.Wrap(err, "Failed to create watcher primary worker promotion")
		}
		log.Info("Watching primary worker promotion is successfully established")

		m.ttlCh, err = m.s.WatchConnectionEnds(ctx, m.primaryID)
		if err != nil {
			return errors.Wrap(err, "Failed to create watcher primary worker connection ends")
		}
		log.Info("Watching primary worker connection ends is successfully established")
	}

	return nil
}

func (m *Manager) tryPromotion(ctx context.Context, _type PromotionType) (bool, error) {
	mu, err := m.s.Mutex(ctx, out.MutexKeyWorker)
	if err != nil {
		return false, errors.Wrap(err, "Failed to create mutex")
	}
	if err := mu.TryLock(ctx); err != nil {
		return false, nil
	}
	defer mu.Unlock(ctx)

	var timestamp time.Time

	switch _type {
	case TTLFailed:

		status, err := m.s.GetWorkerStatus(ctx, m.primaryID)
		if err != nil {
			return false, errors.Wrap(err, "Failed to get primary worker status")
		}
		if status == vo.WorkerStatusExitedTTlFailed {
			return false, nil
		}

		timestamp = time.Now().Add(-time.Second * 5)
		if err := m.s.UpdateWorkerStatusExited(ctx, m.primaryID, vo.WorkerStatusExitedTTlFailed, timestamp); err != nil {
			return false, errors.Wrap(err, "Failed to update primary worker status")
		}

	case Shutdown:
		workers, err := m.s.GetAllWorker(ctx)
		if err != nil {
			return false, errors.Wrap(err, "Failed to get all workers")
		}

		hasPrimary, primaryID := m.hasPrimary(workers)
		if hasPrimary {
			m.primaryID = primaryID
			return false, nil
		}

		timestamp, err = m.s.GetWorkerTimestamp(ctx, m.primaryID)
		if err != nil {
			return false, errors.Wrap(err, "Failed to get primary worker timestamp")
		}
	}

	m.p.UpgradeToStreamingPipe(timestamp)

	m.worker.Status = vo.WorkerStatusPrimary
	if err := m.s.UpdateWorkerStatus(ctx, m.worker.ID, vo.WorkerStatusPrimary); err != nil {
		return false, errors.Wrap(err, "Failed to update worker status to primary")
	}

	close(m.connCh)
	close(m.promCh)
	return true, nil
}

func (m *Manager) Cease() error {
	m.p.Close()
	return nil
}

func (m *Manager) Shutdown() error {
	log.Info("Shutdown is triggered")

	var timestamp = time.Now().Truncate(time.Second).Add(time.Second)

	if m.worker.Status == vo.WorkerStatusPrimary {
		m.p.LockupPipe(timestamp)
	}

	if err := m.s.UpdateWorkerStatusExited(m.ctx, m.worker.ID, vo.WorkerStatusExitedShutdownOccured, timestamp); err != nil {
		return errors.Wrap(err, "Failed to update worker status")
	}

	m.cancel()
	log.Info("Shutdown is successfully completed")
	return nil
}

type PromotionType int

const (
	TTLFailed PromotionType = iota + 1
	Shutdown
)

func (m *Manager) trackChan() {
	m.wg.Add(1)
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return
		case _, ok := <-m.promCh:
			if ok {
				log.Info("Primary worker shutdown detected, trying to promote to primary worker")
				success, err := m.tryPromotion(m.ctx, Shutdown)
				if err != nil {
					panic(errors.Wrap(err, "Failed to promote triggered by primary worker shutdown"))
				}
				if success {
					log.Info("Successfully promoted to primary worker triggered by primary worker shutdown")
					return
				}
			}
		case _, ok := <-m.ttlCh:
			if ok {
				log.Info("Primary worker ttl fail detected, trying to promote to primary worker")
				success, err := m.tryPromotion(m.ctx, TTLFailed)
				if err != nil {
					panic(errors.Wrap(err, "Failed to promote triggered by primary worker ttl fail"))
				}
				if success {
					log.Info("Successfully promoted to primary worker triggered by primary worker ttl fail")
					return
				}
			}
			continue
		}
	}
}

func (m *Manager) OnConnectionFailed() <-chan struct{} {
	return m.connCh
}

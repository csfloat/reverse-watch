package ingestors

import (
	"context"

	"reverse-watch/config"
	"reverse-watch/domain/repository"
	"reverse-watch/ingestors/csfloat"

	"go.uber.org/zap"
)

type ingestor interface {
	Start()
	Stop()
	Done() <-chan struct{}
	IsEnabled() bool
}

type manager struct {
	log       *zap.SugaredLogger
	ingestors []ingestor

	ctx    context.Context
	cancel context.CancelFunc
}

func New(factory repository.Factory, cfg *config.Config, logger *zap.SugaredLogger) *manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &manager{
		log:       logger,
		ingestors: []ingestor{csfloat.NewCSFloatIngestor(ctx, factory, cfg, logger)},
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (m *manager) Start() {
	for _, ingestor := range m.ingestors {
		if !ingestor.IsEnabled() {
			continue
		}
		ingestor.Start()
	}
}

func (m *manager) Stop() {
	for _, ingestor := range m.ingestors {
		ingestor.Stop()
	}
	// Block until each ingestor exits
	for _, ingestor := range m.ingestors {
		<-ingestor.Done()
	}
}

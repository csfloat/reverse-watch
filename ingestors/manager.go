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
}

type manager struct {
	log       *zap.SugaredLogger
	ingestors []ingestor

	ctx    context.Context
	cancel context.CancelFunc
}

func New(factory repository.Factory, cfg *config.Config, logger *zap.SugaredLogger) *manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &manager{
		log:       logger,
		ingestors: make([]ingestor, 0),
		ctx:       ctx,
		cancel:    cancel,
	}

	if cfg.Ingestors.CSFloat.Enable {
		m.ingestors = append(m.ingestors, csfloat.NewCSFloatIngestor(ctx, factory, cfg, logger))
	}
	return m
}

func (m *manager) StartIngestors() {
	for _, ingestor := range m.ingestors {
		ingestor.Start()
	}
}

func (m *manager) Stop() {
	m.cancel()
	for _, ingestor := range m.ingestors {
		ingestor.Stop()
	}
}

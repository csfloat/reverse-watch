package ingestors

import (
	"context"

	"reverse-watch/config"
	"reverse-watch/domain/ingestors"
	"reverse-watch/domain/repository"
	"reverse-watch/ingestors/csfloat"
	"reverse-watch/leader"

	"go.uber.org/zap"
)

type manager struct {
	log       *zap.SugaredLogger
	ingestors map[ingestors.IngestorType]ingestors.Ingestor

	ctx    context.Context
	cancel context.CancelFunc
}

var _ ingestors.Manager = (*manager)(nil)

func New(factory repository.Factory, cfg *config.Config, log *zap.SugaredLogger) ingestors.Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &manager{
		log:       log,
		ingestors: make(map[ingestors.IngestorType]ingestors.Ingestor),
		ctx:       ctx,
		cancel:    cancel,
	}

	elector := leader.New(factory, log)
	if cfg.Ingestors.CSFloat.Enable {
		m.ingestors[ingestors.IngestorTypeCSFloat] = csfloat.NewCSFloatIngestor(ctx, factory, elector, cfg, log)
	}
	return m
}

func (m *manager) StartIngestors() {
	for _, ingestor := range m.ingestors {
		ingestor.Start()
	}
}

func (m *manager) StopIngestors() {
	m.cancel()
	for _, ingestor := range m.ingestors {
		ingestor.Stop()
	}
}

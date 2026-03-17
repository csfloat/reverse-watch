package ingestors

type Ingestor interface {
	Start()
	Stop()
}

type Manager interface {
	StartIngestors()
	StopIngestors()
}

type IngestorType string

const (
	IngestorTypeCSFloat IngestorType = "csfloat"
)

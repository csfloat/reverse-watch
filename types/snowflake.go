package types

import (
	"sync"
	"time"

	"reverse-watch/errors"
)

// January 1, 2019 00:00:0000 in milliseconds
const epoch uint64 = 1546300800000

var (
	once             sync.Once
	defaultGenerator *SnowflakeGenerator
)

type Snowflake uint64

type Parts struct {
	// Timestamp is the first 41 (+1 top zero bit) bits and represents the millisecond-level timestamp
	Timestamp uint64
	// WorkerID is the next 5 bits
	WorkerID uint8
	// ProcessID is the next 5 bits
	ProcessID uint8
	// Sequence is the last 12 bits and represents snowflakes generated within the same millisecond
	Sequence uint16
}

type SnowflakeGenerator struct {
	mutex     sync.Mutex
	lastTime  uint64
	workerID  uint8
	processID uint8
	sequence  uint16
}

func InitSnowflakeGenerator(workerID uint8, processID uint8) {
	once.Do(func() {
		defaultGenerator = &SnowflakeGenerator{
			workerID:  workerID,
			processID: processID,
		}
	})
}

func genSnowflakeWithParts(parts Parts) (Snowflake, error) {
	if parts.Timestamp < epoch {
		return 0, errors.New(errors.InternalServerError, "snowflake's timestamp cannot be before epoch")
	}

	timestamp := parts.Timestamp - epoch

	workerId := uint64(parts.WorkerID) & 0x1F
	processId := uint64(parts.ProcessID) & 0x1F
	seq := uint64(parts.Sequence) & 0xFFF

	var snowflake uint64

	snowflake |= timestamp << 22
	snowflake |= workerId << 17
	snowflake |= processId << 12
	snowflake |= seq

	return Snowflake(snowflake), nil
}

func (g *SnowflakeGenerator) generate() (Snowflake, error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	now := uint64(time.Now().UnixMilli())
	if now < g.lastTime {
		return 0, errors.Newf(errors.InternalServerError, nil, "clock moved backwards by %d ms", g.lastTime-now)
	}

	if now == g.lastTime {
		// Same millisecond -- increment sequence
		g.sequence = (g.sequence + 1) & 0xFFF
		if g.sequence == 0 {
			return 0, errors.Newf(errors.InternalServerError, nil, "sequence exhausted for millisecond %d", now)
		}
	} else {
		g.lastTime = now
		g.sequence = 0
	}

	return genSnowflakeWithParts(Parts{
		Timestamp: now,
		WorkerID:  g.workerID,
		ProcessID: g.processID,
		Sequence:  g.sequence,
	})
}

func GenSnowflake() (Snowflake, error) {
	return defaultGenerator.generate()
}

func ParseSnowflake(snowflake Snowflake) Parts {
	return Parts{
		Timestamp: uint64(snowflake>>22) + epoch,
		WorkerID:  uint8((snowflake >> 17) & 0x1F),
		ProcessID: uint8((snowflake >> 12) & 0x1F),
		Sequence:  uint16(snowflake & 0xFFF),
	}
}

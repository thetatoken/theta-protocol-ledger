package consensus

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/ukulele/common"

	"github.com/spf13/viper"
)

const channelBufferSize = 10

// EpochManager runs its own goroutine to manage epoch for engine. It is not thread-safe.
type EpochManager struct {
	e      Engine
	epoch  uint32
	ticker *time.Ticker
	reset  chan uint32
	C      chan uint32 // Channel for announcing new epoches.

	// Lifecycle.
	mu      *sync.Mutex
	wg      *sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

// NewEpochManager creates a new instance of EpochManager.
func NewEpochManager() *EpochManager {
	return &EpochManager{
		epoch: 0,
		reset: make(chan uint32, channelBufferSize),
		C:     make(chan uint32, channelBufferSize),

		mu: &sync.Mutex{},
		wg: &sync.WaitGroup{},
	}
}

// Init intializes a EpochManager.
func (m *EpochManager) Init(e Engine) {
	m.e = e
}

func (m *EpochManager) resetTimer() {
	if m.ticker != nil {
		m.ticker.Stop()
	}
	m.ticker = time.NewTicker(time.Duration(viper.GetInt(common.CfgConsensusMaxEpochLength)) * time.Second)
}

// Start is the main goroutine loop to handle timeout and newEpoch.
func (m *EpochManager) Start(ctx context.Context) {
	m.resetTimer()
	c, cancel := context.WithCancel(ctx)
	m.ctx = c
	m.cancel = cancel

	m.wg.Add(1)
	go m.mainLoop()
}

// Stop notifies all epoch manager's goroutines to stop without blocking.
func (m *EpochManager) Stop() {
	m.cancel()
}

// Wait blocks until all epoch manager's goroutines have finished.
func (m *EpochManager) Wait() {
	m.wg.Wait()
}

func (m *EpochManager) mainLoop() {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			if m.ticker != nil {
				m.ticker.Stop()
			}

			m.mu.Lock()
			m.stopped = true
			m.mu.Unlock()
			return
		case <-m.reset:
			log.WithFields(log.Fields{"id": m.e.ID(), "m.epoch": m.epoch}).Debug("Proactively moved to new epoch")
			m.resetTimer()
		case <-m.ticker.C:
			log.WithFields(log.Fields{"id": m.e.ID(), "m.epoch": m.epoch, "newEpoch": m.epoch + 1}).Debug("Timed out. Moving to new epoch")
			m.mu.Lock()
			m.epoch++
			m.resetTimer()
			m.mu.Unlock()
			m.C <- m.epoch
		}
	}
}

// SetEpoch notifies the EpochManager to advance to given epoch. This call is non-blocking.
func (m *EpochManager) SetEpoch(epoch uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		return
	}
	m.epoch = epoch
	select {
	case m.reset <- epoch:
	default:
	}
}

// GetEpoch returns the current epoch.
func (m *EpochManager) GetEpoch() uint32 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.epoch
}

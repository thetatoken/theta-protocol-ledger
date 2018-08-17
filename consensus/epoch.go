package consensus

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/ukulele/common"

	"github.com/spf13/viper"
)

// EpochManager runs its own goroutine to manage epoch for engine. It is not thread-safe.
type EpochManager struct {
	e      Engine
	epoch  uint32
	ticker *time.Ticker
	reset  chan uint32
	C      chan uint32 // Channel for announcing new epoches
}

// NewEpochManager creates a new instance of EpochManager.
func NewEpochManager() *EpochManager {
	return &EpochManager{
		epoch: 0,
		reset: make(chan uint32),
		C:     make(chan uint32),
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
	m.ticker = time.NewTicker(time.Duration(viper.GetInt(common.CfgConsesusMaxEpochLength)) * time.Second)
}

func (m *EpochManager) setEpoch(newEpoch uint32) {
	m.epoch = newEpoch
	m.C <- m.epoch
}

// Start is the main goroutine loop to handle timeout and newEpoch.
func (m *EpochManager) Start(ctx context.Context) {
	m.resetTimer()
	for {
		select {
		case <-ctx.Done():
			if m.ticker != nil {
				m.ticker.Stop()
			}
			return
		case newEpoch := <-m.reset:
			log.WithFields(log.Fields{"id": m.e.ID(), "m.epoch": m.epoch, "newEpoch": newEpoch}).Debug("Proactively moving to new epoch")
			m.epoch = newEpoch
			m.resetTimer()
		case <-m.ticker.C:
			log.WithFields(log.Fields{"id": m.e.ID(), "m.epoch": m.epoch, "newEpoch": m.epoch + 1}).Debug("Timed out. Moving to new epoch")
			m.epoch++
			m.C <- m.epoch
			m.resetTimer()
		}
	}
}

// SetEpoch notifies the EpochManager to advance to given epoch. This call is non-blocking.
func (m *EpochManager) SetEpoch(epoch uint32) {
	m.reset <- epoch
}

package consensus

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/util"
)

// EpochManager runs its own goroutine to manage epoch for engine. It is not thread-safe.
type EpochManager struct {
	e      Engine
	height uint32
	ticker *time.Ticker
	reset  chan uint32
	C      chan uint32 // Channel for announcing new heights
}

// NewEpochManager creates a new instance of EpochManager.
func NewEpochManager() *EpochManager {
	return &EpochManager{
		height: 0,
		reset:  make(chan uint32),
		C:      make(chan uint32),
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
	m.ticker = time.NewTicker(time.Duration(viper.GetInt(util.CfgConsesusMaxEpochLength)) * time.Second)
}

func (m *EpochManager) setHeight(newHeight uint32) {
	m.height = newHeight
	m.C <- m.height
}

// Start is the main goroutine loop to handle timeout and newHeight.
func (m *EpochManager) Start(ctx context.Context) {
	m.resetTimer()
	for {
		select {
		case <-ctx.Done():
			if m.ticker != nil {
				m.ticker.Stop()
			}
			return
		case newHeight := <-m.reset:
			log.WithFields(log.Fields{"id": m.e.ID(), "m.height": m.height, "newHeight": newHeight}).Debug("Proactively moving to new height")
			m.height = newHeight
			m.resetTimer()
		case <-m.ticker.C:
			log.WithFields(log.Fields{"id": m.e.ID(), "m.height": m.height, "newHeight": m.height + 1}).Debug("Timed out. Moving to new height")
			m.height++
			m.C <- m.height
			m.resetTimer()
		}
	}
}

// SetHeight notifies the EpochManager to advance to given height. This call is non-blocking.
func (m *EpochManager) SetHeight(height uint32) {
	m.reset <- height
}

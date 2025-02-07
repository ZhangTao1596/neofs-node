package main

import (
	"sync"

	nmClient "github.com/nspcc-dev/neofs-node/pkg/morph/client/netmap"
	"github.com/nspcc-dev/neofs-node/pkg/morph/timer"
)

type (
	// EigenTrustDuration is a structure that provides duration of one
	// eigen trust iteration round in blocks for block timer.
	EigenTrustDuration struct {
		sync.Mutex

		nm  *nmClient.Client
		val uint32
	}
)

// NewEigenTrustDuration returns instance of EigenTrustDuration.
func NewEigenTrustDuration(nm *nmClient.Client) *EigenTrustDuration {
	return &EigenTrustDuration{
		nm: nm,
	}
}

// Value returns number of blocks between two iterations of EigenTrust
// calculation. This value is not changed between `Update` calls.
func (e *EigenTrustDuration) Value() (uint32, error) {
	e.Lock()
	defer e.Unlock()

	if e.val == 0 {
		e.update()
	}

	return e.val, nil
}

// Update function recalculate duration of EigenTrust iteration based on
// NeoFS epoch duration and amount of iteration rounds from global config.
func (e *EigenTrustDuration) Update() {
	e.Lock()
	defer e.Unlock()

	e.update()
}

func (e *EigenTrustDuration) update() {
	iterationAmount, err := e.nm.EigenTrustIterations()
	if err != nil {
		return
	}

	epochDuration, err := e.nm.EpochDuration()
	if err != nil {
		return
	}

	e.val = uint32(epochDuration / iterationAmount)
}

func startBlockTimers(c *cfg) {
	for i := range c.cfgMorph.blockTimers {
		if err := c.cfgMorph.blockTimers[i].Reset(); err != nil {
			fatalOnErr(err)
		}
	}
}

func tickBlockTimers(c *cfg) {
	for i := range c.cfgMorph.blockTimers {
		c.cfgMorph.blockTimers[i].Tick()
	}
}

func newEigenTrustIterTimer(c *cfg, it *EigenTrustDuration, handler timer.BlockTickHandler) {
	c.cfgMorph.eigenTrustTimer = timer.NewBlockTimer(
		it.Value,
		handler,
	)

	c.cfgMorph.blockTimers = append(c.cfgMorph.blockTimers, c.cfgMorph.eigenTrustTimer)
}

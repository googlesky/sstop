//go:build darwin

package platform

import (
	"github.com/googlesky/sstop/internal/model"
)

// DarwinPlatform collects network data using netstat and lsof on macOS.
type DarwinPlatform struct{}

// NewPlatform creates a new macOS platform collector.
func NewPlatform() (Platform, error) {
	return &DarwinPlatform{}, nil
}

func (p *DarwinPlatform) Collect() ([]MappedSocket, []model.InterfaceStats, error) {
	// TODO: implement in Phase 3
	return nil, nil, nil
}

func (p *DarwinPlatform) Close() error {
	return nil
}

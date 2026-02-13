//go:build linux

package platform

import (
	"github.com/googlesky/sstop/internal/model"
)

// LinuxPlatform collects network data using netlink and /proc.
type LinuxPlatform struct{}

// NewPlatform creates a new Linux platform collector.
func NewPlatform() (Platform, error) {
	return &LinuxPlatform{}, nil
}

func (p *LinuxPlatform) Collect() ([]MappedSocket, []model.InterfaceStats, error) {
	// TODO: implement in Phase 2
	return nil, nil, nil
}

func (p *LinuxPlatform) Close() error {
	return nil
}

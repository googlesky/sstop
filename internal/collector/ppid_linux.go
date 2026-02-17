//go:build linux

package collector

import "github.com/googlesky/sstop/internal/platform"

func readPPID(pid uint32) uint32 {
	return platform.ReadPPID(pid)
}

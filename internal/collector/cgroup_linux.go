//go:build linux

package collector

import "github.com/googlesky/sstop/internal/platform"

func readCgroup(pid uint32) (containerID, serviceName string) {
	info := platform.ReadCgroup(pid)
	return info.ContainerID, info.ServiceName
}

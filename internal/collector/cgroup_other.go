//go:build !linux

package collector

func readCgroup(_ uint32) (containerID, serviceName string) {
	return "", ""
}

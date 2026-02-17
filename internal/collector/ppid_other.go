//go:build !linux

package collector

func readPPID(_ uint32) uint32 {
	return 0
}

package platform

import (
	"fmt"
	"net"

	"github.com/googlesky/sstop/internal/model"
)

// MappedSocket is a socket with its owning process info already resolved.
type MappedSocket struct {
	model.Socket
	PID         uint32
	ProcessName string
	Cmdline     string
}

// Platform abstracts OS-specific network data collection.
type Platform interface {
	// Collect returns all current sockets (with process info) and interface stats.
	Collect() (sockets []MappedSocket, ifaces []model.InterfaceStats, err error)

	// Close releases any OS resources.
	Close() error
}

// SocketKey uniquely identifies a socket for delta tracking across polls.
// Cross-platform: does not use inode.
type SocketKey struct {
	Proto   model.Protocol
	SrcAddr string // "ip:port"
	DstAddr string // "ip:port"
}

// MakeSocketKey builds a SocketKey from a MappedSocket.
func MakeSocketKey(s *MappedSocket) SocketKey {
	return SocketKey{
		Proto:   s.Proto,
		SrcAddr: formatAddr(s.SrcIP, s.SrcPort),
		DstAddr: formatAddr(s.DstIP, s.DstPort),
	}
}

func formatAddr(ip net.IP, port uint16) string {
	if ip == nil || ip.IsUnspecified() {
		return fmt.Sprintf("*:%d", port)
	}
	return model.AddrPort(ip, port)
}

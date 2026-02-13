package model

import (
	"fmt"
	"net"
	"time"
)

// Protocol represents a network protocol (TCP/UDP).
type Protocol uint8

const (
	ProtoTCP Protocol = iota
	ProtoUDP
)

func (p Protocol) String() string {
	switch p {
	case ProtoTCP:
		return "TCP"
	case ProtoUDP:
		return "UDP"
	default:
		return "???"
	}
}

// SocketState represents a TCP connection state.
type SocketState uint8

const (
	StateUnknown     SocketState = iota
	StateEstablished             // TCP_ESTABLISHED
	StateSynSent                 // TCP_SYN_SENT
	StateSynRecv                 // TCP_SYN_RECV
	StateFinWait1                // TCP_FIN_WAIT1
	StateFinWait2                // TCP_FIN_WAIT2
	StateTimeWait                // TCP_TIME_WAIT
	StateClose                   // TCP_CLOSE
	StateCloseWait               // TCP_CLOSE_WAIT
	StateLastAck                 // TCP_LAST_ACK
	StateListen                  // TCP_LISTEN
	StateClosing                 // TCP_CLOSING
)

var stateNames = [...]string{
	StateUnknown:     "UNKNOWN",
	StateEstablished: "ESTABLISHED",
	StateSynSent:     "SYN_SENT",
	StateSynRecv:     "SYN_RECV",
	StateFinWait1:    "FIN_WAIT1",
	StateFinWait2:    "FIN_WAIT2",
	StateTimeWait:    "TIME_WAIT",
	StateClose:       "CLOSE",
	StateCloseWait:   "CLOSE_WAIT",
	StateLastAck:     "LAST_ACK",
	StateListen:      "LISTEN",
	StateClosing:     "CLOSING",
}

func (s SocketState) String() string {
	if int(s) < len(stateNames) {
		return stateNames[s]
	}
	return "UNKNOWN"
}

// Socket represents a single network socket with byte counters.
type Socket struct {
	Proto   Protocol
	SrcIP   net.IP
	SrcPort uint16
	DstIP   net.IP
	DstPort uint16
	State   SocketState
	Inode   uint64 // Linux only, 0 on macOS

	// Byte counters (cumulative)
	BytesSent uint64
	BytesRecv uint64
}

// AddrPort returns "ip:port" string for an address.
func AddrPort(ip net.IP, port uint16) string {
	if ip4 := ip.To4(); ip4 != nil {
		return fmt.Sprintf("%s:%d", ip4, port)
	}
	return fmt.Sprintf("[%s]:%d", ip, port)
}

// ProcessInfo holds info about a single process.
type ProcessInfo struct {
	PID     uint32
	Name    string
	Cmdline string
	UID     uint32
}

// Connection represents a single connection with bandwidth info.
type Connection struct {
	Proto    Protocol
	SrcIP    net.IP
	SrcPort  uint16
	DstIP    net.IP
	DstPort  uint16
	State    SocketState
	UpRate   float64 // bytes/sec
	DownRate float64 // bytes/sec

	// Resolved remote hostname (empty if not resolved yet)
	RemoteHost string
}

// ListenPort represents a port a process is listening on.
type ListenPort struct {
	Proto Protocol
	IP    net.IP
	Port  uint16
}

// ProcessSummary aggregates network info for a single process.
type ProcessSummary struct {
	PID         uint32
	Name        string
	Cmdline     string
	UpRate      float64 // bytes/sec aggregate
	DownRate    float64 // bytes/sec aggregate
	Connections []Connection
	ListenPorts []ListenPort
	ConnCount   int
	ListenCount int
}

// InterfaceStats holds per-interface byte counters.
type InterfaceStats struct {
	Name      string
	BytesRecv uint64
	BytesSent uint64
}

// Snapshot is an immutable point-in-time view of all network activity.
type Snapshot struct {
	Timestamp   time.Time
	Processes   []ProcessSummary
	Interfaces  []InterfaceStats
	TotalUp     float64 // bytes/sec
	TotalDown   float64 // bytes/sec
}

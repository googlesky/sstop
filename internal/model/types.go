package model

import (
	"fmt"
	"net"
	"strings"
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
	Proto   Protocol    `json:"proto"`
	SrcIP   net.IP      `json:"src_ip"`
	SrcPort uint16      `json:"src_port"`
	DstIP   net.IP      `json:"dst_ip"`
	DstPort uint16      `json:"dst_port"`
	State   SocketState `json:"state"`
	Inode   uint64      `json:"inode,omitempty"` // Linux only, 0 on macOS

	// Byte counters (cumulative)
	BytesSent uint64 `json:"bytes_sent"`
	BytesRecv uint64 `json:"bytes_recv"`
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
	PID     uint32 `json:"pid"`
	Name    string `json:"name"`
	Cmdline string `json:"cmdline"`
	UID     uint32 `json:"uid"`
}

// Connection represents a single connection with bandwidth info.
type Connection struct {
	Proto    Protocol      `json:"proto"`
	SrcIP    net.IP        `json:"src_ip"`
	SrcPort  uint16        `json:"src_port"`
	DstIP    net.IP        `json:"dst_ip"`
	DstPort  uint16        `json:"dst_port"`
	State    SocketState   `json:"state"`
	UpRate   float64       `json:"up_rate"`   // bytes/sec
	DownRate float64       `json:"down_rate"` // bytes/sec
	Age      time.Duration `json:"age"`       // how long the connection has been tracked

	// Resolved remote hostname (empty if not resolved yet)
	RemoteHost string `json:"remote_host,omitempty"`

	// Service name (e.g. HTTPS, SSH, DNS)
	Service string `json:"service,omitempty"`
}

// ListenPort represents a port a process is listening on.
type ListenPort struct {
	Proto Protocol `json:"proto"`
	IP    net.IP   `json:"ip"`
	Port  uint16   `json:"port"`
}

// ProcessSummary aggregates network info for a single process.
type ProcessSummary struct {
	PID         uint32       `json:"pid"`
	PPID        uint32       `json:"ppid,omitempty"`
	Name        string       `json:"name"`
	Cmdline     string       `json:"cmdline"`
	UpRate      float64      `json:"up_rate"`  // bytes/sec aggregate
	DownRate    float64      `json:"down_rate"` // bytes/sec aggregate
	Connections []Connection `json:"connections"`
	ListenPorts []ListenPort `json:"listen_ports"`
	ConnCount   int          `json:"conn_count"`
	ListenCount int          `json:"listen_count"`

	// Cumulative bytes (populated when cumulative tracking is active)
	CumUp   uint64 `json:"cum_up,omitempty"`
	CumDown uint64 `json:"cum_down,omitempty"`

	// Container/service group info
	ContainerID string `json:"container_id,omitempty"` // Docker/Podman short ID
	ServiceName string `json:"service_name,omitempty"` // systemd service name

	// Sparkline history (total rate = up+down, chronological, oldest first)
	RateHistory []float64 `json:"-"`
}

// InterfaceStats holds per-interface byte counters and rates.
type InterfaceStats struct {
	Name      string  `json:"name"`
	BytesRecv uint64  `json:"bytes_recv"`
	BytesSent uint64  `json:"bytes_sent"`
	RecvRate  float64 `json:"recv_rate"` // bytes/sec (computed by collector)
	SendRate  float64 `json:"send_rate"` // bytes/sec (computed by collector)
}

// RemoteHostSummary aggregates bandwidth by remote host across all processes.
type RemoteHostSummary struct {
	Host      string   `json:"host"`       // hostname or IP string
	IP        net.IP   `json:"ip"`         // raw IP
	UpRate    float64  `json:"up_rate"`    // bytes/sec
	DownRate  float64  `json:"down_rate"`  // bytes/sec
	ConnCount int      `json:"conn_count"` // number of connections
	Processes []string `json:"processes"`  // process names connected to this host
	Country   string   `json:"country,omitempty"` // country code (e.g. "US")
}

// ListenPortEntry is a system-wide listening port with its owning process.
type ListenPortEntry struct {
	Proto   Protocol `json:"proto"`
	IP      net.IP   `json:"ip"`
	Port    uint16   `json:"port"`
	PID     uint32   `json:"pid"`
	Process string   `json:"process"`
	Cmdline string   `json:"cmdline"`
}

// SessionStats holds cumulative session statistics (shown on exit).
type SessionStats struct {
	Duration   time.Duration
	TotalUp    uint64              // cumulative bytes uploaded
	TotalDown  uint64              // cumulative bytes downloaded
	TopProcess []ProcessCumulative // top 5 by total bytes
}

// ProcessCumulative tracks cumulative bytes for a single process.
type ProcessCumulative struct {
	PID       uint32
	Name      string
	BytesUp   uint64
	BytesDown uint64
}

// Summary returns a formatted string for terminal display on exit.
func (s SessionStats) Summary() string {
	if s.TotalUp == 0 && s.TotalDown == 0 && len(s.TopProcess) == 0 {
		return ""
	}

	var b strings.Builder
	dur := s.Duration.Truncate(time.Second)
	b.WriteString(fmt.Sprintf("\nsstop session: %s\n", dur))
	b.WriteString(fmt.Sprintf("Total: ▲ %s  ▼ %s\n", fmtBytes(s.TotalUp), fmtBytes(s.TotalDown)))

	if len(s.TopProcess) > 0 {
		b.WriteString("Top processes:\n")
		for i, p := range s.TopProcess {
			if p.BytesUp == 0 && p.BytesDown == 0 {
				continue
			}
			b.WriteString(fmt.Sprintf("  %d. %-16s ▲ %-10s ▼ %s\n",
				i+1, p.Name, fmtBytes(p.BytesUp), fmtBytes(p.BytesDown)))
		}
	}
	return b.String()
}

func fmtBytes(b uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case b >= TB:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(TB))
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// Snapshot is an immutable point-in-time view of all network activity.
type Snapshot struct {
	Timestamp    time.Time            `json:"timestamp"`
	Processes    []ProcessSummary     `json:"processes"`
	Interfaces   []InterfaceStats     `json:"interfaces"`
	RemoteHosts  []RemoteHostSummary  `json:"remote_hosts"`
	ListenPorts  []ListenPortEntry    `json:"listen_ports"`
	TotalUp      float64              `json:"total_up"`   // bytes/sec
	TotalDown    float64              `json:"total_down"` // bytes/sec

	// Total rate history for header sparkline (up+down combined)
	TotalRateHistory []float64 `json:"-"`

	// Active interface name (empty = all)
	ActiveIface string `json:"-"`
}

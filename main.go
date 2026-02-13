package main

import (
	"fmt"
	"os"

	"github.com/googlesky/sstop/internal/model"
	"github.com/googlesky/sstop/internal/platform"
)

func main() {
	p, err := platform.NewPlatform()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init platform: %v\n", err)
		os.Exit(1)
	}
	defer p.Close()

	sockets, ifaces, err := p.Collect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "collect failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Sockets: %d, Interfaces: %d\n", len(sockets), len(ifaces))
	for i, s := range sockets {
		if i >= 20 {
			fmt.Printf("... and %d more\n", len(sockets)-20)
			break
		}
		fmt.Printf("  [%s] %s -> %s  state=%s  pid=%d (%s)  sent=%d recv=%d\n",
			s.Proto, formatAddrDisplay(s.Socket), formatDstDisplay(s.Socket),
			s.State, s.PID, s.ProcessName, s.BytesSent, s.BytesRecv)
	}
	for _, iface := range ifaces {
		fmt.Printf("  iface %s: recv=%d sent=%d\n", iface.Name, iface.BytesRecv, iface.BytesSent)
	}
}

func formatAddrDisplay(s model.Socket) string {
	return model.AddrPort(s.SrcIP, s.SrcPort)
}

func formatDstDisplay(s model.Socket) string {
	return model.AddrPort(s.DstIP, s.DstPort)
}

//go:build linux

package platform

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/googlesky/sstop/internal/model"
)

// InodeInfo maps an inode to its process.
type InodeInfo struct {
	PID     uint32
	Name    string
	Cmdline string
}

// ScanProcesses walks /proc to build a map of socket inode → process info.
func ScanProcesses() (map[uint64]InodeInfo, error) {
	result := make(map[uint64]InodeInfo)

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.ParseUint(entry.Name(), 10, 32)
		if err != nil {
			continue // not a PID directory
		}

		pidU32 := uint32(pid)
		fdDir := filepath.Join("/proc", entry.Name(), "fd")

		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue // permission denied or process exited
		}

		// Lazy-load process info only if we find socket inodes.
		var info *InodeInfo

		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err != nil {
				continue
			}

			// Socket links look like "socket:[12345]"
			if !strings.HasPrefix(link, "socket:[") {
				continue
			}
			inodeStr := link[8 : len(link)-1]
			inode, err := strconv.ParseUint(inodeStr, 10, 64)
			if err != nil {
				continue
			}

			if info == nil {
				name, cmdline := readProcessInfo(pidU32)
				info = &InodeInfo{
					PID:     pidU32,
					Name:    name,
					Cmdline: cmdline,
				}
			}

			result[inode] = *info
		}
	}

	return result, nil
}

// readProcessInfo reads /proc/<pid>/comm and /proc/<pid>/cmdline.
func readProcessInfo(pid uint32) (name, cmdline string) {
	pidStr := strconv.FormatUint(uint64(pid), 10)

	// Read comm (process name, max 16 chars)
	if data, err := os.ReadFile(filepath.Join("/proc", pidStr, "comm")); err == nil {
		name = strings.TrimSpace(string(data))
	}

	// Read cmdline (null-separated)
	if data, err := os.ReadFile(filepath.Join("/proc", pidStr, "cmdline")); err == nil {
		// Replace null bytes with spaces
		cmdline = string(bytes.ReplaceAll(data, []byte{0}, []byte{' '}))
		cmdline = strings.TrimSpace(cmdline)
	}

	if name == "" {
		name = "?"
	}
	return
}

// ReadPPID reads the parent PID of a process from /proc/<pid>/stat.
func ReadPPID(pid uint32) uint32 {
	pidStr := strconv.FormatUint(uint64(pid), 10)
	data, err := os.ReadFile(filepath.Join("/proc", pidStr, "stat"))
	if err != nil {
		return 0
	}

	// /proc/<pid>/stat format: pid (comm) state ppid ...
	// comm can contain spaces and parens, so find last ')' first
	s := string(data)
	lastParen := strings.LastIndex(s, ")")
	if lastParen < 0 || lastParen+2 >= len(s) {
		return 0
	}

	// After ") " comes: state ppid ...
	fields := strings.Fields(s[lastParen+2:])
	if len(fields) < 2 {
		return 0
	}

	ppid, err := strconv.ParseUint(fields[1], 10, 32)
	if err != nil {
		return 0
	}
	return uint32(ppid)
}

// ParseNetDev reads /proc/net/dev and returns interface stats.
func ParseNetDev() ([]model.InterfaceStats, error) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, fmt.Errorf("open /proc/net/dev: %w", err)
	}
	defer f.Close()

	var result []model.InterfaceStats
	scanner := bufio.NewScanner(f)

	// Skip header lines
	for i := 0; i < 2 && scanner.Scan(); i++ {
	}

	for scanner.Scan() {
		line := scanner.Text()

		// Format: "  iface: recv_bytes packets ... | send_bytes packets ..."
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}

		ifaceName := strings.TrimSpace(line[:colonIdx])
		fields := strings.Fields(line[colonIdx+1:])
		if len(fields) < 10 {
			continue
		}

		// Skip loopback
		if ifaceName == "lo" {
			continue
		}

		recvBytes, _ := strconv.ParseUint(fields[0], 10, 64)
		sentBytes, _ := strconv.ParseUint(fields[8], 10, 64)

		result = append(result, model.InterfaceStats{
			Name:      ifaceName,
			BytesRecv: recvBytes,
			BytesSent: sentBytes,
		})
	}

	return result, scanner.Err()
}

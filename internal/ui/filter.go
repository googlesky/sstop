package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/googlesky/sstop/internal/model"
)

// Filter represents a parsed filter expression.
type Filter struct {
	raw      string
	key      string  // empty for plain text search
	op       string  // ":", ">", "<"
	value    string
	numValue float64
}

// ParseFilter parses a filter string into a Filter.
// Supports: plain text, key:value, key>value, key<value.
func ParseFilter(input string) Filter {
	input = strings.TrimSpace(input)
	if input == "" {
		return Filter{}
	}

	// Try to find operator
	for _, op := range []string{">", "<", ":"} {
		idx := strings.Index(input, op)
		if idx > 0 {
			key := strings.ToLower(input[:idx])
			value := input[idx+1:]
			f := Filter{raw: input, key: key, op: op, value: value}
			if op == ">" || op == "<" {
				f.numValue = parseSize(value)
			}
			return f
		}
	}

	// Plain text search
	return Filter{raw: input}
}

// IsEmpty returns true if the filter matches everything.
func (f Filter) IsEmpty() bool {
	return f.raw == ""
}

// Match returns true if the process matches the filter.
func (f Filter) Match(proc *model.ProcessSummary) bool {
	if f.raw == "" {
		return true
	}

	// Plain text search (backward compatible)
	if f.key == "" {
		lower := strings.ToLower(f.raw)
		return strings.Contains(strings.ToLower(proc.Name), lower) ||
			strings.Contains(strings.ToLower(proc.Cmdline), lower) ||
			strings.Contains(fmt.Sprintf("%d", proc.PID), f.raw)
	}

	switch f.key {
	case "port":
		return f.matchPort(proc)
	case "up":
		return f.matchNumeric(proc.UpRate)
	case "down":
		return f.matchNumeric(proc.DownRate)
	case "proto":
		return f.matchProto(proc)
	case "host":
		return f.matchHost(proc)
	case "conns":
		return f.matchNumeric(float64(proc.ConnCount))
	case "listen":
		return f.matchListen(proc)
	case "svc", "service":
		return f.matchService(proc)
	case "group":
		return f.matchGroup(proc)
	default:
		// Unknown key — fall back to plain text search
		lower := strings.ToLower(f.raw)
		return strings.Contains(strings.ToLower(proc.Name), lower) ||
			strings.Contains(strings.ToLower(proc.Cmdline), lower)
	}
}

func (f Filter) matchPort(proc *model.ProcessSummary) bool {
	port, err := strconv.ParseUint(f.value, 10, 16)
	if err != nil {
		return false
	}
	p := uint16(port)
	for _, c := range proc.Connections {
		if c.SrcPort == p || c.DstPort == p {
			return true
		}
	}
	for _, lp := range proc.ListenPorts {
		if lp.Port == p {
			return true
		}
	}
	return false
}

func (f Filter) matchNumeric(val float64) bool {
	switch f.op {
	case ">":
		return val > f.numValue
	case "<":
		return val < f.numValue
	case ":":
		return val > f.numValue
	}
	return false
}

func (f Filter) matchProto(proc *model.ProcessSummary) bool {
	want := strings.ToUpper(f.value)
	for _, c := range proc.Connections {
		if c.Proto.String() == want {
			return true
		}
	}
	return false
}

func (f Filter) matchHost(proc *model.ProcessSummary) bool {
	lower := strings.ToLower(f.value)
	for _, c := range proc.Connections {
		if strings.Contains(strings.ToLower(c.RemoteHost), lower) {
			return true
		}
		if c.DstIP != nil && strings.Contains(c.DstIP.String(), f.value) {
			return true
		}
	}
	return false
}

func (f Filter) matchListen(proc *model.ProcessSummary) bool {
	v := strings.ToLower(f.value)
	if v == "true" || v == "yes" || v == "1" {
		return proc.ListenCount > 0
	}
	return proc.ListenCount == 0
}

func (f Filter) matchService(proc *model.ProcessSummary) bool {
	lower := strings.ToLower(f.value)
	for _, c := range proc.Connections {
		if strings.Contains(strings.ToLower(c.Service), lower) {
			return true
		}
	}
	return false
}

func (f Filter) matchGroup(proc *model.ProcessSummary) bool {
	lower := strings.ToLower(f.value)
	// Match against container ID or service name
	if proc.ContainerID != "" && strings.Contains(strings.ToLower(proc.ContainerID), lower) {
		return true
	}
	if proc.ServiceName != "" && strings.Contains(strings.ToLower(proc.ServiceName), lower) {
		return true
	}
	// Match "other" for ungrouped processes
	if lower == "other" && proc.ContainerID == "" && proc.ServiceName == "" {
		return true
	}
	return false
}

// parseSize parses a human-readable size string like "1M", "100K", "1G".
func parseSize(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	multiplier := 1.0
	last := s[len(s)-1]
	switch last {
	case 'k', 'K':
		multiplier = 1024
		s = s[:len(s)-1]
	case 'm', 'M':
		multiplier = 1024 * 1024
		s = s[:len(s)-1]
	case 'g', 'G':
		multiplier = 1024 * 1024 * 1024
		s = s[:len(s)-1]
	case 't', 'T':
		multiplier = 1024 * 1024 * 1024 * 1024
		s = s[:len(s)-1]
	}

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val * multiplier
}

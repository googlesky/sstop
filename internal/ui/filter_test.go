package ui

import (
	"net"
	"testing"

	"github.com/googlesky/sstop/internal/model"
)

func testProc() model.ProcessSummary {
	return model.ProcessSummary{
		PID:     1234,
		Name:    "firefox",
		Cmdline: "/usr/bin/firefox",
		UpRate:  1024 * 1024, // 1 MB/s
		DownRate: 2 * 1024 * 1024,
		Connections: []model.Connection{
			{
				Proto:      model.ProtoTCP,
				SrcIP:      net.ParseIP("192.168.1.5"),
				SrcPort:    54321,
				DstIP:      net.ParseIP("142.250.80.46"),
				DstPort:    443,
				State:      model.StateEstablished,
				RemoteHost: "google.com",
				Service:    "HTTPS",
			},
			{
				Proto:      model.ProtoUDP,
				SrcIP:      net.ParseIP("192.168.1.5"),
				SrcPort:    12345,
				DstIP:      net.ParseIP("8.8.8.8"),
				DstPort:    53,
				State:      model.StateEstablished,
				RemoteHost: "dns.google",
				Service:    "DNS",
			},
		},
		ListenPorts: []model.ListenPort{
			{Proto: model.ProtoTCP, IP: net.IPv4zero, Port: 8080},
		},
		ConnCount:   2,
		ListenCount: 1,
	}
}

func TestFilterPlainText(t *testing.T) {
	p := testProc()
	f := ParseFilter("firefox")
	if !f.Match(&p) {
		t.Error("plain text 'firefox' should match")
	}
	f = ParseFilter("chrome")
	if f.Match(&p) {
		t.Error("plain text 'chrome' should not match")
	}
}

func TestFilterPort(t *testing.T) {
	p := testProc()
	f := ParseFilter("port:443")
	if !f.Match(&p) {
		t.Error("port:443 should match")
	}
	f = ParseFilter("port:8080")
	if !f.Match(&p) {
		t.Error("port:8080 should match (listen port)")
	}
	f = ParseFilter("port:9999")
	if f.Match(&p) {
		t.Error("port:9999 should not match")
	}
}

func TestFilterUp(t *testing.T) {
	p := testProc()
	f := ParseFilter("up>500K")
	if !f.Match(&p) {
		t.Error("up>500K should match (1 MB/s)")
	}
	f = ParseFilter("up>2M")
	if f.Match(&p) {
		t.Error("up>2M should not match (1 MB/s)")
	}
}

func TestFilterDown(t *testing.T) {
	p := testProc()
	f := ParseFilter("down>1M")
	if !f.Match(&p) {
		t.Error("down>1M should match (2 MB/s)")
	}
}

func TestFilterProto(t *testing.T) {
	p := testProc()
	f := ParseFilter("proto:tcp")
	if !f.Match(&p) {
		t.Error("proto:tcp should match")
	}
	f = ParseFilter("proto:udp")
	if !f.Match(&p) {
		t.Error("proto:udp should match")
	}
}

func TestFilterHost(t *testing.T) {
	p := testProc()
	f := ParseFilter("host:google")
	if !f.Match(&p) {
		t.Error("host:google should match")
	}
	f = ParseFilter("host:amazon")
	if f.Match(&p) {
		t.Error("host:amazon should not match")
	}
}

func TestFilterConns(t *testing.T) {
	p := testProc()
	f := ParseFilter("conns>1")
	if !f.Match(&p) {
		t.Error("conns>1 should match (2 conns)")
	}
	f = ParseFilter("conns>5")
	if f.Match(&p) {
		t.Error("conns>5 should not match (2 conns)")
	}
}

func TestFilterListen(t *testing.T) {
	p := testProc()
	f := ParseFilter("listen:true")
	if !f.Match(&p) {
		t.Error("listen:true should match")
	}

	noListen := model.ProcessSummary{Name: "curl"}
	f = ParseFilter("listen:true")
	if f.Match(&noListen) {
		t.Error("listen:true should not match process with no listen ports")
	}
}

func TestFilterService(t *testing.T) {
	p := testProc()
	f := ParseFilter("svc:https")
	if !f.Match(&p) {
		t.Error("svc:https should match")
	}
	f = ParseFilter("svc:ssh")
	if f.Match(&p) {
		t.Error("svc:ssh should not match")
	}
}

func TestFilterEmpty(t *testing.T) {
	f := ParseFilter("")
	if !f.IsEmpty() {
		t.Error("empty filter should be empty")
	}
	p := testProc()
	if !f.Match(&p) {
		t.Error("empty filter should match everything")
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"100", 100},
		{"1K", 1024},
		{"1k", 1024},
		{"1M", 1024 * 1024},
		{"1.5M", 1.5 * 1024 * 1024},
		{"1G", 1024 * 1024 * 1024},
		{"", 0},
		{"abc", 0},
	}
	for _, tt := range tests {
		got := parseSize(tt.input)
		if got != tt.want {
			t.Errorf("parseSize(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

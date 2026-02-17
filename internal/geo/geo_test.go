package geo

import (
	"net"
	"testing"
)

func TestLookupPrivate(t *testing.T) {
	tests := []struct {
		ip   string
		code string
	}{
		{"192.168.1.1", "LAN"},
		{"10.0.0.1", "LAN"},
		{"172.16.0.1", "LAN"},
		{"172.31.255.255", "LAN"},
		{"100.64.0.1", "LAN"},
		{"127.0.0.1", "LO"},
	}
	for _, tt := range tests {
		info := Lookup(net.ParseIP(tt.ip))
		if info.Code != tt.code {
			t.Errorf("Lookup(%s) = %q, want %q", tt.ip, info.Code, tt.code)
		}
	}
}

func TestLookupGoogle(t *testing.T) {
	// Google DNS
	info := Lookup(net.ParseIP("8.8.8.8"))
	if info.Code != "US" {
		t.Errorf("Lookup(8.8.8.8) = %q, want US", info.Code)
	}
}

func TestLookupCloudflare(t *testing.T) {
	info := Lookup(net.ParseIP("1.1.1.1"))
	if info.Code != "US" {
		t.Errorf("Lookup(1.1.1.1) = %q, want US", info.Code)
	}
}

func TestLookupUnknown(t *testing.T) {
	// Some random IP that might not be in our ranges
	info := Lookup(net.ParseIP("169.254.1.1"))
	if info.Code != "LAN" {
		t.Errorf("Lookup(169.254.1.1) = %q, want LAN (link-local)", info.Code)
	}
}

func TestLookupNil(t *testing.T) {
	info := Lookup(nil)
	if info.Code != "" {
		t.Errorf("Lookup(nil) = %q, want empty", info.Code)
	}
}

func TestCountryFlag(t *testing.T) {
	flag := countryFlag("US")
	if flag != "🇺🇸" {
		t.Errorf("countryFlag(US) = %q, want 🇺🇸", flag)
	}
	flag = countryFlag("VN")
	if flag != "🇻🇳" {
		t.Errorf("countryFlag(VN) = %q, want 🇻🇳", flag)
	}
}

func TestFormat(t *testing.T) {
	c := CountryInfo{Code: "US", Flag: "🇺🇸"}
	if c.Format() != "🇺🇸 US" {
		t.Errorf("Format() = %q, want '🇺🇸 US'", c.Format())
	}

	empty := CountryInfo{}
	if empty.Format() != "" {
		t.Errorf("empty Format() = %q, want empty", empty.Format())
	}
}

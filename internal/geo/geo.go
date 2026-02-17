package geo

import (
	"encoding/binary"
	"net"
)

// CountryInfo holds a country code and flag emoji.
type CountryInfo struct {
	Code string // e.g. "US"
	Flag string // e.g. "🇺🇸"
}

// Lookup returns the country for an IP address.
// Returns empty CountryInfo for unknown IPs.
func Lookup(ip net.IP) CountryInfo {
	if ip == nil {
		return CountryInfo{}
	}

	// Normalize to IPv4
	ip4 := ip.To4()
	if ip4 == nil {
		// IPv6 — not supported in simple mode
		return CountryInfo{}
	}

	// Check private/special IPs first
	if isPrivate(ip4) {
		return CountryInfo{Code: "LAN", Flag: "🏠"}
	}
	if ip4.IsLoopback() {
		return CountryInfo{Code: "LO", Flag: "🏠"}
	}
	if ip4.IsMulticast() {
		return CountryInfo{Code: "MC", Flag: "📡"}
	}

	// Find the most specific (smallest) matching range
	ipNum := ipToUint32(ip4)
	var bestMatch *ipRange
	for i := range ipRanges {
		r := &ipRanges[i]
		if ipNum >= r.start && ipNum <= r.end {
			if bestMatch == nil || (r.end-r.start) < (bestMatch.end-bestMatch.start) {
				bestMatch = r
			}
		}
	}

	if bestMatch != nil {
		return CountryInfo{Code: bestMatch.country, Flag: countryFlag(bestMatch.country)}
	}

	return CountryInfo{}
}

// Format returns "🇺🇸 US" style string, or "" if unknown.
func (c CountryInfo) Format() string {
	if c.Code == "" {
		return ""
	}
	return c.Flag + " " + c.Code
}

func isPrivate(ip net.IP) bool {
	privateRanges := []struct {
		network string
		mask    string
	}{
		{"10.0.0.0", "255.0.0.0"},
		{"172.16.0.0", "255.240.0.0"},
		{"192.168.0.0", "255.255.0.0"},
		{"100.64.0.0", "255.192.0.0"}, // CGNAT
		{"169.254.0.0", "255.255.0.0"}, // Link-local
	}
	for _, r := range privateRanges {
		network := net.ParseIP(r.network).To4()
		mask := net.IPMask(net.ParseIP(r.mask).To4())
		if network.Mask(mask).Equal(ip.Mask(mask)) {
			return true
		}
	}
	return false
}

func ipToUint32(ip net.IP) uint32 {
	ip4 := ip.To4()
	if ip4 == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip4)
}

func countryFlag(code string) string {
	if len(code) != 2 {
		return "🌐"
	}
	// Regional indicator symbols: 🇦 = U+1F1E6
	r1 := rune(code[0]-'A') + 0x1F1E6
	r2 := rune(code[1]-'A') + 0x1F1E6
	return string([]rune{r1, r2})
}

// ipRange represents a range of IPs belonging to a country.
type ipRange struct {
	start   uint32
	end     uint32
	country string
}

// ipRanges is a sorted list of IP ranges for major cloud/CDN providers and countries.
// Covers the most common IP ranges seen in network traffic.
var ipRanges = []ipRange{
	// Google (US)
	{ipToU32(8, 8, 4, 0), ipToU32(8, 8, 8, 255), "US"},
	{ipToU32(8, 34, 208, 0), ipToU32(8, 35, 207, 255), "US"},
	{ipToU32(34, 0, 0, 0), ipToU32(34, 127, 255, 255), "US"},
	{ipToU32(35, 184, 0, 0), ipToU32(35, 199, 255, 255), "US"},
	{ipToU32(64, 233, 160, 0), ipToU32(64, 233, 191, 255), "US"},
	{ipToU32(66, 102, 0, 0), ipToU32(66, 102, 15, 255), "US"},
	{ipToU32(66, 249, 64, 0), ipToU32(66, 249, 95, 255), "US"},
	{ipToU32(72, 14, 192, 0), ipToU32(72, 14, 255, 255), "US"},
	{ipToU32(74, 125, 0, 0), ipToU32(74, 125, 255, 255), "US"},
	{ipToU32(142, 250, 0, 0), ipToU32(142, 251, 255, 255), "US"},
	{ipToU32(172, 217, 0, 0), ipToU32(172, 217, 255, 255), "US"},
	{ipToU32(173, 194, 0, 0), ipToU32(173, 194, 255, 255), "US"},
	{ipToU32(209, 85, 128, 0), ipToU32(209, 85, 255, 255), "US"},
	{ipToU32(216, 58, 192, 0), ipToU32(216, 58, 223, 255), "US"},

	// Amazon AWS (US)
	{ipToU32(3, 0, 0, 0), ipToU32(3, 127, 255, 255), "US"},
	{ipToU32(13, 32, 0, 0), ipToU32(13, 35, 255, 255), "US"},
	{ipToU32(13, 224, 0, 0), ipToU32(13, 255, 255, 255), "US"},
	{ipToU32(52, 0, 0, 0), ipToU32(52, 95, 255, 255), "US"},
	{ipToU32(54, 64, 0, 0), ipToU32(54, 95, 255, 255), "US"},
	{ipToU32(54, 144, 0, 0), ipToU32(54, 255, 255, 255), "US"},

	// Microsoft/Azure (US)
	{ipToU32(13, 64, 0, 0), ipToU32(13, 107, 255, 255), "US"},
	{ipToU32(20, 0, 0, 0), ipToU32(20, 63, 255, 255), "US"},
	{ipToU32(40, 64, 0, 0), ipToU32(40, 127, 255, 255), "US"},
	{ipToU32(52, 96, 0, 0), ipToU32(52, 191, 255, 255), "US"},
	{ipToU32(104, 40, 0, 0), ipToU32(104, 47, 255, 255), "US"},
	{ipToU32(204, 79, 195, 0), ipToU32(204, 79, 197, 255), "US"},

	// Cloudflare (US)
	{ipToU32(1, 0, 0, 0), ipToU32(1, 1, 1, 255), "US"},
	{ipToU32(104, 16, 0, 0), ipToU32(104, 31, 255, 255), "US"},
	{ipToU32(172, 64, 0, 0), ipToU32(172, 71, 255, 255), "US"},
	{ipToU32(188, 114, 96, 0), ipToU32(188, 114, 99, 255), "US"},
	{ipToU32(198, 41, 128, 0), ipToU32(198, 41, 255, 255), "US"},

	// Meta/Facebook (US)
	{ipToU32(31, 13, 24, 0), ipToU32(31, 13, 31, 255), "US"},
	{ipToU32(157, 240, 0, 0), ipToU32(157, 240, 255, 255), "US"},
	{ipToU32(179, 60, 192, 0), ipToU32(179, 60, 195, 255), "US"},

	// Akamai (US)
	{ipToU32(23, 0, 0, 0), ipToU32(23, 79, 255, 255), "US"},
	{ipToU32(104, 64, 0, 0), ipToU32(104, 127, 255, 255), "US"},

	// Apple (US)
	{ipToU32(17, 0, 0, 0), ipToU32(17, 255, 255, 255), "US"},

	// Germany (DE) — common ranges
	{ipToU32(5, 1, 0, 0), ipToU32(5, 1, 127, 255), "DE"},
	{ipToU32(46, 0, 0, 0), ipToU32(46, 0, 255, 255), "DE"},
	{ipToU32(78, 46, 0, 0), ipToU32(78, 47, 255, 255), "DE"},
	{ipToU32(85, 13, 128, 0), ipToU32(85, 13, 255, 255), "DE"},
	{ipToU32(195, 50, 140, 0), ipToU32(195, 50, 143, 255), "DE"},

	// UK (GB)
	{ipToU32(2, 16, 0, 0), ipToU32(2, 31, 255, 255), "GB"},
	{ipToU32(5, 62, 0, 0), ipToU32(5, 63, 255, 255), "GB"},
	{ipToU32(51, 0, 0, 0), ipToU32(51, 15, 255, 255), "GB"},

	// France (FR)
	{ipToU32(2, 0, 0, 0), ipToU32(2, 15, 255, 255), "FR"},
	{ipToU32(5, 39, 0, 0), ipToU32(5, 39, 127, 255), "FR"},
	{ipToU32(51, 68, 0, 0), ipToU32(51, 79, 255, 255), "FR"},
	{ipToU32(91, 134, 0, 0), ipToU32(91, 134, 255, 255), "FR"},

	// Netherlands (NL)
	{ipToU32(5, 2, 0, 0), ipToU32(5, 2, 255, 255), "NL"},
	{ipToU32(31, 3, 0, 0), ipToU32(31, 3, 255, 255), "NL"},
	{ipToU32(37, 48, 0, 0), ipToU32(37, 63, 255, 255), "NL"},
	{ipToU32(178, 162, 0, 0), ipToU32(178, 162, 255, 255), "NL"},

	// Japan (JP)
	{ipToU32(1, 0, 16, 0), ipToU32(1, 0, 31, 255), "JP"},
	{ipToU32(27, 0, 0, 0), ipToU32(27, 15, 255, 255), "JP"},
	{ipToU32(36, 2, 0, 0), ipToU32(36, 3, 255, 255), "JP"},
	{ipToU32(49, 212, 0, 0), ipToU32(49, 213, 255, 255), "JP"},
	{ipToU32(103, 5, 140, 0), ipToU32(103, 5, 143, 255), "JP"},
	{ipToU32(133, 0, 0, 0), ipToU32(133, 255, 255, 255), "JP"},
	{ipToU32(210, 0, 0, 0), ipToU32(210, 255, 255, 255), "JP"},

	// China (CN)
	{ipToU32(1, 0, 1, 0), ipToU32(1, 0, 3, 255), "CN"},
	{ipToU32(14, 0, 0, 0), ipToU32(14, 31, 255, 255), "CN"},
	{ipToU32(36, 0, 0, 0), ipToU32(36, 1, 255, 255), "CN"},
	{ipToU32(42, 0, 0, 0), ipToU32(42, 127, 255, 255), "CN"},
	{ipToU32(58, 0, 0, 0), ipToU32(58, 63, 255, 255), "CN"},
	{ipToU32(101, 0, 0, 0), ipToU32(101, 127, 255, 255), "CN"},
	{ipToU32(106, 0, 0, 0), ipToU32(106, 127, 255, 255), "CN"},
	{ipToU32(110, 0, 0, 0), ipToU32(110, 255, 255, 255), "CN"},
	{ipToU32(111, 0, 0, 0), ipToU32(111, 255, 255, 255), "CN"},
	{ipToU32(112, 0, 0, 0), ipToU32(112, 127, 255, 255), "CN"},
	{ipToU32(114, 0, 0, 0), ipToU32(114, 127, 255, 255), "CN"},
	{ipToU32(116, 0, 0, 0), ipToU32(116, 127, 255, 255), "CN"},
	{ipToU32(119, 0, 0, 0), ipToU32(119, 63, 255, 255), "CN"},
	{ipToU32(120, 0, 0, 0), ipToU32(120, 127, 255, 255), "CN"},
	{ipToU32(121, 0, 0, 0), ipToU32(121, 127, 255, 255), "CN"},
	{ipToU32(122, 0, 0, 0), ipToU32(122, 127, 255, 255), "CN"},
	{ipToU32(123, 0, 0, 0), ipToU32(123, 127, 255, 255), "CN"},
	{ipToU32(124, 0, 0, 0), ipToU32(124, 127, 255, 255), "CN"},
	{ipToU32(125, 0, 0, 0), ipToU32(125, 127, 255, 255), "CN"},
	{ipToU32(180, 76, 0, 0), ipToU32(180, 76, 255, 255), "CN"},
	{ipToU32(182, 0, 0, 0), ipToU32(182, 127, 255, 255), "CN"},
	{ipToU32(183, 0, 0, 0), ipToU32(183, 255, 255, 255), "CN"},
	{ipToU32(202, 96, 0, 0), ipToU32(202, 111, 255, 255), "CN"},
	{ipToU32(218, 0, 0, 0), ipToU32(218, 127, 255, 255), "CN"},
	{ipToU32(220, 0, 0, 0), ipToU32(220, 255, 255, 255), "CN"},
	{ipToU32(221, 0, 0, 0), ipToU32(221, 255, 255, 255), "CN"},
	{ipToU32(222, 0, 0, 0), ipToU32(222, 255, 255, 255), "CN"},
	{ipToU32(223, 0, 0, 0), ipToU32(223, 255, 255, 255), "CN"},

	// South Korea (KR)
	{ipToU32(1, 11, 0, 0), ipToU32(1, 11, 255, 255), "KR"},
	{ipToU32(14, 32, 0, 0), ipToU32(14, 63, 255, 255), "KR"},
	{ipToU32(27, 96, 0, 0), ipToU32(27, 127, 255, 255), "KR"},
	{ipToU32(39, 0, 0, 0), ipToU32(39, 15, 255, 255), "KR"},
	{ipToU32(58, 64, 0, 0), ipToU32(58, 79, 255, 255), "KR"},
	{ipToU32(112, 128, 0, 0), ipToU32(112, 191, 255, 255), "KR"},
	{ipToU32(175, 192, 0, 0), ipToU32(175, 223, 255, 255), "KR"},
	{ipToU32(211, 0, 0, 0), ipToU32(211, 63, 255, 255), "KR"},

	// India (IN)
	{ipToU32(14, 96, 0, 0), ipToU32(14, 143, 255, 255), "IN"},
	{ipToU32(27, 56, 0, 0), ipToU32(27, 63, 255, 255), "IN"},
	{ipToU32(43, 224, 0, 0), ipToU32(43, 255, 255, 255), "IN"},
	{ipToU32(49, 32, 0, 0), ipToU32(49, 47, 255, 255), "IN"},
	{ipToU32(103, 0, 0, 0), ipToU32(103, 5, 139, 255), "IN"},
	{ipToU32(117, 192, 0, 0), ipToU32(117, 255, 255, 255), "IN"},

	// Russia (RU)
	{ipToU32(5, 3, 0, 0), ipToU32(5, 3, 255, 255), "RU"},
	{ipToU32(5, 8, 0, 0), ipToU32(5, 8, 255, 255), "RU"},
	{ipToU32(31, 13, 0, 0), ipToU32(31, 13, 23, 255), "RU"},
	{ipToU32(46, 8, 0, 0), ipToU32(46, 8, 255, 255), "RU"},
	{ipToU32(77, 88, 0, 0), ipToU32(77, 88, 63, 255), "RU"},
	{ipToU32(87, 240, 0, 0), ipToU32(87, 240, 255, 255), "RU"},
	{ipToU32(93, 158, 0, 0), ipToU32(93, 158, 255, 255), "RU"},
	{ipToU32(95, 163, 0, 0), ipToU32(95, 163, 255, 255), "RU"},
	{ipToU32(185, 32, 0, 0), ipToU32(185, 32, 127, 255), "RU"},
	{ipToU32(213, 180, 0, 0), ipToU32(213, 180, 255, 255), "RU"},

	// Brazil (BR)
	{ipToU32(45, 160, 0, 0), ipToU32(45, 175, 255, 255), "BR"},
	{ipToU32(131, 0, 0, 0), ipToU32(131, 0, 255, 255), "BR"},
	{ipToU32(177, 0, 0, 0), ipToU32(177, 127, 255, 255), "BR"},
	{ipToU32(179, 0, 0, 0), ipToU32(179, 60, 191, 255), "BR"},
	{ipToU32(186, 192, 0, 0), ipToU32(186, 255, 255, 255), "BR"},
	{ipToU32(187, 0, 0, 0), ipToU32(187, 127, 255, 255), "BR"},
	{ipToU32(189, 0, 0, 0), ipToU32(189, 127, 255, 255), "BR"},
	{ipToU32(200, 0, 0, 0), ipToU32(200, 255, 255, 255), "BR"},
	{ipToU32(201, 0, 0, 0), ipToU32(201, 63, 255, 255), "BR"},

	// Australia (AU)
	{ipToU32(1, 0, 4, 0), ipToU32(1, 0, 7, 255), "AU"},
	{ipToU32(1, 40, 0, 0), ipToU32(1, 47, 255, 255), "AU"},
	{ipToU32(27, 32, 0, 0), ipToU32(27, 55, 255, 255), "AU"},
	{ipToU32(43, 224, 0, 0), ipToU32(43, 239, 255, 255), "AU"},
	{ipToU32(49, 176, 0, 0), ipToU32(49, 191, 255, 255), "AU"},
	{ipToU32(101, 128, 0, 0), ipToU32(101, 191, 255, 255), "AU"},
	{ipToU32(103, 128, 0, 0), ipToU32(103, 143, 255, 255), "AU"},
	{ipToU32(203, 0, 0, 0), ipToU32(203, 63, 255, 255), "AU"},

	// Canada (CA)
	{ipToU32(24, 48, 0, 0), ipToU32(24, 63, 255, 255), "CA"},
	{ipToU32(67, 68, 0, 0), ipToU32(67, 71, 255, 255), "CA"},
	{ipToU32(99, 224, 0, 0), ipToU32(99, 255, 255, 255), "CA"},
	{ipToU32(142, 0, 0, 0), ipToU32(142, 3, 255, 255), "CA"},
	{ipToU32(192, 206, 0, 0), ipToU32(192, 206, 255, 255), "CA"},
	{ipToU32(199, 7, 0, 0), ipToU32(199, 7, 255, 255), "CA"},

	// Singapore (SG)
	{ipToU32(1, 32, 0, 0), ipToU32(1, 39, 255, 255), "SG"},
	{ipToU32(13, 212, 0, 0), ipToU32(13, 215, 255, 255), "SG"},
	{ipToU32(27, 124, 0, 0), ipToU32(27, 125, 255, 255), "SG"},
	{ipToU32(43, 128, 0, 0), ipToU32(43, 159, 255, 255), "SG"},
	{ipToU32(49, 128, 0, 0), ipToU32(49, 143, 255, 255), "SG"},
	{ipToU32(52, 74, 0, 0), ipToU32(52, 77, 255, 255), "SG"},
	{ipToU32(54, 169, 0, 0), ipToU32(54, 169, 255, 255), "SG"},
	{ipToU32(103, 6, 0, 0), ipToU32(103, 7, 255, 255), "SG"},
	{ipToU32(175, 41, 128, 0), ipToU32(175, 41, 191, 255), "SG"},

	// Vietnam (VN)
	{ipToU32(1, 52, 0, 0), ipToU32(1, 55, 255, 255), "VN"},
	{ipToU32(14, 160, 0, 0), ipToU32(14, 191, 255, 255), "VN"},
	{ipToU32(27, 64, 0, 0), ipToU32(27, 79, 255, 255), "VN"},
	{ipToU32(42, 112, 0, 0), ipToU32(42, 119, 255, 255), "VN"},
	{ipToU32(43, 239, 0, 0), ipToU32(43, 239, 255, 255), "VN"},
	{ipToU32(49, 156, 0, 0), ipToU32(49, 159, 255, 255), "VN"},
	{ipToU32(58, 186, 0, 0), ipToU32(58, 187, 255, 255), "VN"},
	{ipToU32(103, 1, 0, 0), ipToU32(103, 1, 255, 255), "VN"},
	{ipToU32(113, 160, 0, 0), ipToU32(113, 191, 255, 255), "VN"},
	{ipToU32(115, 72, 0, 0), ipToU32(115, 79, 255, 255), "VN"},
	{ipToU32(171, 224, 0, 0), ipToU32(171, 255, 255, 255), "VN"},
	{ipToU32(180, 148, 0, 0), ipToU32(180, 148, 255, 255), "VN"},
	{ipToU32(203, 113, 0, 0), ipToU32(203, 113, 255, 255), "VN"},
	{ipToU32(203, 162, 0, 0), ipToU32(203, 162, 255, 255), "VN"},
	{ipToU32(210, 86, 0, 0), ipToU32(210, 86, 255, 255), "VN"},
}

func ipToU32(a, b, c, d byte) uint32 {
	return uint32(a)<<24 | uint32(b)<<16 | uint32(c)<<8 | uint32(d)
}


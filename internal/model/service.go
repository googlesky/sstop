package model

// serviceMap maps well-known ports to service names.
var serviceMap = map[uint16]string{
	20:    "FTP-D",
	21:    "FTP",
	22:    "SSH",
	23:    "TELNET",
	25:    "SMTP",
	53:    "DNS",
	67:    "DHCP",
	68:    "DHCP",
	80:    "HTTP",
	110:   "POP3",
	123:   "NTP",
	143:   "IMAP",
	161:   "SNMP",
	162:   "SNMP",
	389:   "LDAP",
	443:   "HTTPS",
	445:   "SMB",
	465:   "SMTPS",
	514:   "SYSLOG",
	587:   "SMTP",
	636:   "LDAPS",
	853:   "DoT",
	993:   "IMAPS",
	995:   "POP3S",
	1080:  "SOCKS",
	1194:  "OVPN",
	1433:  "MSSQL",
	1434:  "MSSQL",
	1521:  "ORACLE",
	1883:  "MQTT",
	2049:  "NFS",
	3306:  "MYSQL",
	3389:  "RDP",
	5432:  "PGSQL",
	5672:  "AMQP",
	5900:  "VNC",
	6379:  "REDIS",
	6443:  "K8S",
	8080:  "HTTP-A",
	8443:  "HTTPS",
	8883:  "MQTTS",
	9090:  "PROM",
	9092:  "KAFKA",
	9200:  "ELAST",
	9300:  "ELAST",
	11211: "MEMCD",
	27017: "MONGO",
	5353:  "MDNS",
	8888:  "HTTP-A",
}

// ServiceName returns the service name for a port.
// Checks DstPort first, then SrcPort. Returns "" if unknown.
func ServiceName(dstPort, srcPort uint16) string {
	if s, ok := serviceMap[dstPort]; ok {
		return s
	}
	if s, ok := serviceMap[srcPort]; ok {
		return s
	}
	return ""
}

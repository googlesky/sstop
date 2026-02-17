//go:build linux

package platform

import (
	"fmt"
	"os"
	"strings"
)

// CgroupInfo holds parsed cgroup information for a process.
type CgroupInfo struct {
	ContainerID string // Docker/Podman container short ID (12 chars)
	ServiceName string // systemd service name (e.g. "nginx.service")
}

// ReadCgroup reads /proc/<pid>/cgroup and extracts container/service info.
func ReadCgroup(pid uint32) CgroupInfo {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return CgroupInfo{}
	}
	return parseCgroup(string(data))
}

// parseCgroup parses cgroup file content and extracts container/service info.
func parseCgroup(content string) CgroupInfo {
	var info CgroupInfo

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: hierarchy-ID:controller-list:cgroup-path
		// e.g. "0::/system.slice/nginx.service"
		// e.g. "0::/system.slice/docker-abc123...xyz.scope"
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		cgPath := parts[2]

		// Docker: path contains /docker/<container-id>
		if info.ContainerID == "" {
			if id := extractDockerID(cgPath); id != "" {
				info.ContainerID = id
			}
		}

		// Podman: path contains /libpod-<container-id>
		if info.ContainerID == "" {
			if id := extractPodmanID(cgPath); id != "" {
				info.ContainerID = id
			}
		}

		// Systemd service: path contains /<name>.service
		if info.ServiceName == "" {
			if svc := extractSystemdService(cgPath); svc != "" {
				info.ServiceName = svc
			}
		}
	}

	return info
}

// extractDockerID extracts short container ID from docker cgroup paths.
// Handles:
//   - /docker/<id>
//   - /docker-<id>.scope
//   - /system.slice/docker-<id>.scope
func extractDockerID(cgPath string) string {
	// /docker/<full-id>
	if idx := strings.Index(cgPath, "/docker/"); idx >= 0 {
		id := cgPath[idx+len("/docker/"):]
		// Strip anything after the id (sub-paths)
		if slashIdx := strings.Index(id, "/"); slashIdx >= 0 {
			id = id[:slashIdx]
		}
		return shortID(id)
	}

	// docker-<full-id>.scope
	for _, seg := range strings.Split(cgPath, "/") {
		if strings.HasPrefix(seg, "docker-") && strings.HasSuffix(seg, ".scope") {
			id := strings.TrimPrefix(seg, "docker-")
			id = strings.TrimSuffix(id, ".scope")
			return shortID(id)
		}
	}

	return ""
}

// extractPodmanID extracts short container ID from podman cgroup paths.
// Handles: /libpod-<id>
func extractPodmanID(cgPath string) string {
	for _, seg := range strings.Split(cgPath, "/") {
		if strings.HasPrefix(seg, "libpod-") {
			id := strings.TrimPrefix(seg, "libpod-")
			// May have suffix like .scope
			if dotIdx := strings.Index(id, "."); dotIdx >= 0 {
				id = id[:dotIdx]
			}
			return shortID(id)
		}
	}
	return ""
}

// extractSystemdService extracts service name from systemd cgroup paths.
// Handles:
//   - /system.slice/nginx.service
//   - /user.slice/user-1000.slice/...
func extractSystemdService(cgPath string) string {
	for _, seg := range strings.Split(cgPath, "/") {
		if strings.HasSuffix(seg, ".service") {
			// Skip docker-*.service as those are container entries
			if strings.HasPrefix(seg, "docker-") {
				continue
			}
			return seg
		}
	}
	return ""
}

// shortID returns first 12 chars of a container ID (standard short format).
func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

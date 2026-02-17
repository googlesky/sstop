//go:build linux

package platform

import (
	"testing"
)

func TestParseCgroup_DockerScope(t *testing.T) {
	// Docker container via systemd scope: docker-<full-id>.scope
	content := "0::/system.slice/docker-abc123def456789.scope"
	info := parseCgroup(content)

	if info.ContainerID != "abc123def456" {
		t.Errorf("ContainerID = %q, want %q", info.ContainerID, "abc123def456")
	}
	// docker- prefix services should be skipped
	if info.ServiceName != "" {
		t.Errorf("ServiceName = %q, want empty (docker- prefix should be skipped)", info.ServiceName)
	}
}

func TestParseCgroup_DockerPath(t *testing.T) {
	// Docker container via /docker/<id> path
	content := "0::/docker/abc123def456789aaa/"
	info := parseCgroup(content)

	if info.ContainerID != "abc123def456" {
		t.Errorf("ContainerID = %q, want %q", info.ContainerID, "abc123def456")
	}
	if info.ServiceName != "" {
		t.Errorf("ServiceName = %q, want empty", info.ServiceName)
	}
}

func TestParseCgroup_Podman(t *testing.T) {
	// Podman container via libpod- prefix
	content := "0::/machine.slice/libpod-xyz789abc123.scope"
	info := parseCgroup(content)

	if info.ContainerID != "xyz789abc123" {
		t.Errorf("ContainerID = %q, want %q", info.ContainerID, "xyz789abc123")
	}
	if info.ServiceName != "" {
		t.Errorf("ServiceName = %q, want empty", info.ServiceName)
	}
}

func TestParseCgroup_SystemdService(t *testing.T) {
	// Systemd service (non-docker)
	content := "0::/system.slice/nginx.service"
	info := parseCgroup(content)

	if info.ContainerID != "" {
		t.Errorf("ContainerID = %q, want empty", info.ContainerID)
	}
	if info.ServiceName != "nginx.service" {
		t.Errorf("ServiceName = %q, want %q", info.ServiceName, "nginx.service")
	}
}

func TestParseCgroup_DockerServiceSkipped(t *testing.T) {
	// docker-*.service should NOT set ServiceName (it's a container entry)
	content := "0::/system.slice/docker-abc.service"
	info := parseCgroup(content)

	if info.ServiceName != "" {
		t.Errorf("ServiceName = %q, want empty (docker- prefix should be skipped)", info.ServiceName)
	}
}

func TestParseCgroup_UserScope(t *testing.T) {
	// User scope — no container, no service
	content := "0::/user.slice/user-1000.slice/session-1.scope"
	info := parseCgroup(content)

	if info.ContainerID != "" {
		t.Errorf("ContainerID = %q, want empty", info.ContainerID)
	}
	if info.ServiceName != "" {
		t.Errorf("ServiceName = %q, want empty", info.ServiceName)
	}
}

func TestParseCgroup_EmptyContent(t *testing.T) {
	info := parseCgroup("")
	if info.ContainerID != "" {
		t.Errorf("ContainerID = %q, want empty", info.ContainerID)
	}
	if info.ServiceName != "" {
		t.Errorf("ServiceName = %q, want empty", info.ServiceName)
	}
}

func TestParseCgroup_MultipleLines(t *testing.T) {
	// Multiple lines: first line has docker container, second line has service
	content := "12:memory:/docker/aabbccddee11223344\n0::/system.slice/nginx.service\n"
	info := parseCgroup(content)

	if info.ContainerID != "aabbccddee11" {
		t.Errorf("ContainerID = %q, want %q", info.ContainerID, "aabbccddee11")
	}
	if info.ServiceName != "nginx.service" {
		t.Errorf("ServiceName = %q, want %q", info.ServiceName, "nginx.service")
	}
}

func TestShortID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{"long id", "abc123def456789", "abc123def456"},
		{"exactly 12", "abc123def456", "abc123def456"},
		{"short id", "abc", "abc"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shortID(tt.id)
			if got != tt.want {
				t.Errorf("shortID(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestExtractDockerID(t *testing.T) {
	tests := []struct {
		name   string
		cgPath string
		want   string
	}{
		{"docker path", "/docker/abc123def456789", "abc123def456"},
		{"docker path with subpath", "/docker/abc123def456789/subpath", "abc123def456"},
		{"docker scope", "/system.slice/docker-abc123def456789.scope", "abc123def456"},
		{"no docker", "/system.slice/nginx.service", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDockerID(tt.cgPath)
			if got != tt.want {
				t.Errorf("extractDockerID(%q) = %q, want %q", tt.cgPath, got, tt.want)
			}
		})
	}
}

func TestExtractPodmanID(t *testing.T) {
	tests := []struct {
		name   string
		cgPath string
		want   string
	}{
		{"libpod scope", "/machine.slice/libpod-xyz789abc123.scope", "xyz789abc123"},
		{"libpod no suffix", "/machine.slice/libpod-xyz789abc123def456", "xyz789abc123"},
		{"no podman", "/system.slice/nginx.service", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPodmanID(tt.cgPath)
			if got != tt.want {
				t.Errorf("extractPodmanID(%q) = %q, want %q", tt.cgPath, got, tt.want)
			}
		})
	}
}

func TestExtractSystemdService(t *testing.T) {
	tests := []struct {
		name   string
		cgPath string
		want   string
	}{
		{"nginx service", "/system.slice/nginx.service", "nginx.service"},
		{"docker service skipped", "/system.slice/docker-abc.service", ""},
		{"no service", "/user.slice/user-1000.slice/session-1.scope", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSystemdService(tt.cgPath)
			if got != tt.want {
				t.Errorf("extractSystemdService(%q) = %q, want %q", tt.cgPath, got, tt.want)
			}
		})
	}
}

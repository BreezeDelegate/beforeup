package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func windowsHost(t *testing.T, endpoint, engineOS, desktopName string) Host {
	t.Helper()
	return Host{
		Run: func(_ context.Context, cmd string, args ...string) ([]byte, error) {
			command := cmd + " " + strings.Join(args, " ")
			switch {
			case strings.HasPrefix(command, "docker compose "):
				return []byte(goodModel), nil
			case strings.HasPrefix(command, "docker context inspect"):
				return []byte(endpoint), nil
			case command == "docker info --format {{.OSType}}":
				if engineOS == "" {
					return nil, errors.New("daemon error secret")
				}
				return []byte(engineOS), nil
			case command == "docker info --format {{.OperatingSystem}}":
				return []byte(desktopName), nil
			case strings.HasPrefix(command, "docker ps --filter label=com.docker.compose.project=example"):
				return []byte(""), nil
			default:
				t.Fatalf("Windows must not call %q", command)
				return nil, errors.New("unexpected command")
			}
		},
		Stat: func(string) (os.FileInfo, error) {
			t.Fatal("Windows must not stat Linux VM bind paths")
			return nil, os.ErrNotExist
		},
		Free: func(string) (uint64, error) {
			t.Fatal("Windows must not inspect Windows disk as Docker VM disk")
			return 0, nil
		},
		Getenv: func(string) string { return "" },
	}
}

func TestWindowsDesktopSkipsHostProbesButComparesRunningState(t *testing.T) {
	h := windowsHost(t, "npipe:////./pipe/docker_engine", "linux", "Docker Desktop")
	r, err := inspectForOS(context.Background(), Options{Compare: true, MinFreeGiB: 2}, h, "windows")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"COMPOSE", "DAEMON", "DESKTOP_VM", "NO_RUNNING_STACK"} {
		if !hasID(r, id) {
			t.Fatalf("missing %s: %+v", id, r)
		}
	}
	for _, id := range []string{"BIND_MISSING", "PORTS", "DISK", "SOCKETS", "RUNNING_MATCH"} {
		if hasID(r, id) {
			t.Fatalf("unsafe %s: %+v", id, r)
		}
	}
	if r.Summary.Blockers != 0 {
		t.Fatalf("unexpected blockers: %+v", r)
	}
}

func TestWindowsRemoteContextOnlyValidatesCompose(t *testing.T) {
	h := windowsHost(t, "npipe:////./pipe/docker_engine", "linux", "Docker Desktop")
	h.Getenv = func(k string) string {
		if k == "DOCKER_HOST" {
			return "tcp://server.example:2376"
		}
		return ""
	}
	r, err := inspectForOS(context.Background(), Options{Compare: true}, h, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if !hasID(r, "CONTEXT") || hasID(r, "DAEMON") || hasID(r, "NO_RUNNING_STACK") {
		t.Fatalf("remote context cannot inspect workstation or running stack: %+v", r)
	}
}

func TestWindowsDaemonUnavailableDoesNotPretendToCompare(t *testing.T) {
	h := windowsHost(t, "npipe:////./pipe/docker_engine", "", "")
	r, err := inspectForOS(context.Background(), Options{Compare: true}, h, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if !hasID(r, "DAEMON") || !hasID(r, "BASELINE_UNKNOWN") || hasID(r, "NO_RUNNING_STACK") {
		t.Fatalf("missing daemon safety markers: %+v", r)
	}
	if strings.Contains(strings.ToLower(strings.Join(func() []string {
		var parts []string
		for _, f := range r.Findings {
			parts = append(parts, f.Message, f.Advice)
		}
		return parts
	}(), " ")), "secret") {
		t.Fatal("leaked daemon error")
	}
}

func TestWindowsContainersAreNotAdvertisedAsSupported(t *testing.T) {
	h := windowsHost(t, "npipe:////./pipe/docker_engine", "windows", "Docker Desktop")
	r, err := inspectForOS(context.Background(), Options{Compare: true}, h, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if !hasID(r, "WINDOWS_ENGINE_UNSUPPORTED") || hasID(r, "NO_RUNNING_STACK") || hasID(r, "RUNNING_MATCH") {
		t.Fatalf("must not compare Windows containers: %+v", r)
	}
}

func TestWindowsDesktopDetectsDeclaredPortCollisionWithoutHostProbes(t *testing.T) {
	const collisionModel = `{"name":"example","services":{"first":{"ports":[{"published":"48093","host_ip":"127.0.0.1","protocol":"tcp"}]},"second":{"ports":[{"published":"48093","host_ip":"127.0.0.1","protocol":"tcp"}]}}}`
	h := windowsHost(t, "npipe:////./pipe/docker_engine", "linux", "Docker Desktop")
	run := h.Run
	h.Run = func(ctx context.Context, cmd string, args ...string) ([]byte, error) {
		if strings.HasPrefix(cmd+" "+strings.Join(args, " "), "docker compose ") {
			return []byte(collisionModel), nil
		}
		return run(ctx, cmd, args...)
	}

	r, err := inspectForOS(context.Background(), Options{}, h, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if !hasID(r, "PORT_COLLISION") || r.Summary.Blockers != 1 {
		t.Fatalf("declared collision must remain a blocker on Docker Desktop: %+v", r)
	}
}

func TestWindowsDeclaredPortCases(t *testing.T) {
	tests := []struct {
		name           string
		secondHost     string
		secondProtocol string
		secondProfile  string
		enabled        bool
		collision      bool
	}{
		{"same host and protocol", "127.0.0.1", "tcp", "", false, true},
		{"different protocol", "127.0.0.1", "udp", "", false, false},
		{"different host IP", "127.0.0.2", "tcp", "", false, false},
		{"inactive profile", "127.0.0.1", "tcp", "debug", false, false},
		{"enabled profile", "127.0.0.1", "tcp", "debug", true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			profile := ""
			if tc.secondProfile != "" {
				profile = `,"profiles":["debug"]`
			}
			raw := `{"name":"example","services":{"first":{"ports":[{"published":"48093","host_ip":"127.0.0.1","protocol":"tcp"}]},"second":{"ports":[{"published":"48093","host_ip":"` + tc.secondHost + `","protocol":"` + tc.secondProtocol + `"}]` + profile + `}}}`
			h := windowsHost(t, "npipe:////./pipe/docker_engine", "linux", "Docker Desktop")
			originalRun := h.Run
			h.Run = func(ctx context.Context, command string, args ...string) ([]byte, error) {
				if command == "docker" && len(args) > 0 && args[0] == "compose" {
					return []byte(raw), nil
				}
				return originalRun(ctx, command, args...)
			}
			var options Options
			if tc.enabled {
				options.Profiles = []string{"debug"}
			}
			r, err := inspectForOS(context.Background(), options, h, "windows")
			if err != nil {
				t.Fatal(err)
			}
			if hasID(r, "PORT_COLLISION") != tc.collision {
				t.Fatalf("incorrect collision detection: %+v", r)
			}
			if hasID(r, "SOCKETS") || hasID(r, "PORTS") || hasID(r, "DISK") || hasID(r, "BIND_MISSING") {
				t.Fatalf("unexpected Windows host probe: %+v", r)
			}
		})
	}
}

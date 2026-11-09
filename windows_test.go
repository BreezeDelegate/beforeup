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

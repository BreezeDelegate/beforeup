package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestDockerDesktopSkipsHostProbesButComparesContainers(t *testing.T) {
	h := fakeHost(`{"name":"example","services":{"api":{"volumes":[{"type":"bind","source":"/fake-host","bind":{"create_host_path":false}}]}}}`)
	old := h.Run
	h.Run = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		c := command + " " + strings.Join(args, " ")
		if strings.Contains(c, "docker info --format {{.OperatingSystem}}") {
			return []byte("Docker Desktop\n"), nil
		}
		if strings.HasPrefix(c, "ss ") || strings.Contains(c, "docker ps --filter publish=") || strings.HasPrefix(c, "sysctl ") {
			t.Fatalf("local host probe on Docker Desktop: %s", c)
		}
		return old(ctx, command, args...)
	}
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil {
		t.Fatal(e)
	}
	if !hasID(r, "DESKTOP_VM") || hasID(r, "BIND_MISSING") || hasID(r, "PORTS") || hasID(r, "DISK") {
		t.Fatalf("Docker Desktop host mismatch: %+v", r)
	}
}
func TestRootlessBindAccessIsNotDeclaredSafe(t *testing.T) {
	h := fakeHost(`{"name":"r","services":{"api":{"volumes":[{"type":"bind","source":"/existent","target":"/data"}]}}}`)
	old := h.Run
	h.Run = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		c := command + " " + strings.Join(args, " ")
		if strings.Contains(c, "docker info --format {{json .SecurityOptions}}") {
			return []byte(`["name=rootless","name=seccomp"]`), nil
		}
		return old(ctx, command, args...)
	}
	h.Stat = func(string) (os.FileInfo, error) { return nil, nil }
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil {
		t.Fatal(e)
	}
	if !hasID(r, "ROOTLESS_BIND_UNVERIFIED") {
		t.Fatalf("rootless bind access was silently assumed: %+v", r)
	}
}
func TestRootlessLowPortPolicyIsAWarningNotBlocker(t *testing.T) {
	h := fakeHost(`{"name":"r","services":{"api":{"ports":[{"published":"80","protocol":"tcp"}]}}}`)
	old := h.Run
	h.Run = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		c := command + " " + strings.Join(args, " ")
		if strings.Contains(c, "docker info --format {{json .SecurityOptions}}") {
			return []byte(`["name=rootless"]`), nil
		}
		if strings.HasPrefix(c, "sysctl -n ") {
			return []byte("1024\n"), nil
		}
		return old(ctx, command, args...)
	}
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil {
		t.Fatal(e)
	}
	if !hasID(r, "ROOTLESS_LOW_PORT") || r.Summary.Blockers > 0 {
		t.Fatalf("low port policy incorrect: %+v", r)
	}
}
func TestRootlessLowPortWithRelaxedSysctl(t *testing.T) {
	h := fakeHost(`{"name":"r","services":{"api":{"ports":[{"published":"80","protocol":"tcp"}]}}}`)
	old := h.Run
	h.Run = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		c := command + " " + strings.Join(args, " ")
		if strings.Contains(c, "docker info --format {{json .SecurityOptions}}") {
			return []byte(`["name=rootless"]`), nil
		}
		if strings.HasPrefix(c, "sysctl -n ") {
			return []byte("0\n"), nil
		}
		return old(ctx, command, args...)
	}
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil {
		t.Fatal(e)
	}
	if hasID(r, "ROOTLESS_LOW_PORT") {
		t.Fatalf("relaxed low-port policy false positive: %+v", r)
	}
}
func TestCrossFamilyDualStackDoesNotClaimCertainPortCollision(t *testing.T) {
	h := fakeHost(`{"name":"dual","services":{"v4":{"ports":[{"host_ip":"0.0.0.0","published":"18129","protocol":"tcp"}]},"v6":{"ports":[{"host_ip":"::","published":"18129","protocol":"tcp"}]}}}`)
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil {
		t.Fatal(e)
	}
	if hasID(r, "PORT_COLLISION") || !hasID(r, "PORT_DUAL_STACK_UNCERTAIN") {
		t.Fatalf("IPv6/V4 certainty overstated: %+v", r)
	}
}
func TestSameFamilyOverlapRemainsBlocker(t *testing.T) {
	h := fakeHost(`{"name":"dual","services":{"v4":{"ports":[{"host_ip":"0.0.0.0","published":"18129","protocol":"tcp"}]},"another":{"ports":[{"host_ip":"127.0.0.1","published":"18129","protocol":"tcp"}]}}}`)
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil {
		t.Fatal(e)
	}
	if !hasID(r, "PORT_COLLISION") {
		t.Fatalf("real IPv4 collision lost: %+v", r)
	}
}
func TestStrictCIReturnsFailureForWarningsOnly(t *testing.T) {
	// The same valid profile-only model emits a warning but no blocker.
	h := fakeHost(`{"name":"only-profile","services":{}}`)
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil || r.Summary.Warnings == 0 || r.Summary.Blockers != 0 {
		t.Fatalf("fixture invalid %+v %v", r, e)
	}
	if reportExitCode(r, nil, false) != 0 || reportExitCode(r, nil, true) != 1 {
		t.Fatalf("strict CI exit code policy incorrect")
	}
	if reportExitCode(r, fmt.Errorf("invalid"), true) != 2 {
		t.Fatal("inspection errors must stay exit code 2")
	}
}

func TestRootlessPortPolicyProbeIsCached(t *testing.T) {
	h := fakeHost(`{"name":"r","services":{"api":{"ports":[{"published":"80","protocol":"tcp"},{"published":"443","protocol":"tcp"}]}}}`)
	old := h.Run
	calls := 0
	h.Run = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		c := command + " " + strings.Join(args, " ")
		if strings.Contains(c, "docker info --format {{json .SecurityOptions}}") {
			return []byte(`["name=rootless"]`), nil
		}
		if strings.HasPrefix(c, "sysctl -n ") {
			calls++
			return []byte("1024\n"), nil
		}
		return old(ctx, command, args...)
	}
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil {
		t.Fatal(e)
	}
	if calls != 1 {
		t.Fatalf("expected one sysctl query for two rootless ports, got %d", calls)
	}
	if n := countFindings(r, "ROOTLESS_LOW_PORT"); n != 2 {
		t.Fatalf("expected two low-port warnings, got %d: %+v", n, r)
	}
}
func countFindings(r Report, id string) int {
	n := 0
	for _, f := range r.Findings {
		if f.ID == id {
			n++
		}
	}
	return n
}

func TestDefaultHostIPPortCollisionIsConfirmed(t *testing.T) {
	raw := `{"name":"dup","services":{"one":{"ports":[{"published":"18129","protocol":"tcp"}]},"two":{"ports":[{"published":"18129","protocol":"tcp"}]}}}`
	r, e := inspect(context.Background(), Options{}, fakeHost(raw))
	if e != nil {
		t.Fatal(e)
	}
	if !hasID(r, "PORT_COLLISION") || hasID(r, "PORT_DUAL_STACK_UNCERTAIN") {
		t.Fatalf("default published host port must collide: %+v", r)
	}
}

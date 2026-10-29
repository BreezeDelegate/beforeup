package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goodModel = `{"name":"example","services":{"web":{"ports":[{"published":"8080","host_ip":"127.0.0.1","protocol":"tcp"}],"volumes":[{"type":"bind","source":"/missing/sample","bind":{"create_host_path":false}}]},"api":{"ports":[],"volumes":[]}}}`

func fakeHost(config string) Host {
	return Host{
		Run: func(_ context.Context, command string, args ...string) ([]byte, error) {
			k := command + " " + strings.Join(args, " ")
			switch {
			case strings.HasPrefix(k, "docker compose "):
				return []byte(config), nil
			case strings.HasPrefix(k, "docker context inspect"):
				return []byte("unix:///var/run/docker.sock\n"), nil
			case strings.HasPrefix(k, "docker info"):
				return []byte("/var/lib/docker\n"), nil
			case strings.HasPrefix(k, "ss -H -ltn"):
				return []byte("LISTEN 0 128 127.0.0.1:8080 0.0.0.0:*\n"), nil
			case strings.HasPrefix(k, "ss -H -lun"):
				return []byte("UNCONN 0 0 0.0.0.0:5353 0.0.0.0:*\n"), nil
			default:
				return nil, fmt.Errorf("unexpected command %q", k)
			}
		},
		Stat:   func(string) (os.FileInfo, error) { return nil, os.ErrNotExist },
		Free:   func(string) (uint64, error) { return 100 * 1073741824, nil },
		Getenv: func(string) string { return "" },
	}
}
func find(r Report, id string) (Finding, bool) {
	for _, f := range r.Findings {
		if f.ID == id {
			return f, true
		}
	}
	return Finding{}, false
}
func TestInspectDetectsRealHostProblems(t *testing.T) {
	r, e := inspect(context.Background(), Options{MinFreeGiB: 2}, fakeHost(goodModel))
	if e != nil {
		t.Fatal(e)
	}
	if r.Project != "example" {
		t.Fatalf("project: %s", r.Project)
	}
	if r.Summary.Blockers != 1 {
		t.Fatalf("expected missing hard bind blocker: %+v", r)
	}
	if _, yes := find(r, "BIND_MISSING"); !yes {
		t.Fatal("missing bind not detected")
	}
	if _, yes := find(r, "PORT_IN_USE"); !yes {
		t.Fatal("host port conflict not detected")
	}
	if r.Summary.Passed < 2 {
		t.Fatal("missing baseline checks")
	}
}
func TestShortBindIsWarningNotBlocker(t *testing.T) {
	raw := `{"name":"x","services":{"api":{"volumes":[{"type":"bind","source":"/missing/sample","bind":{}}]}}}`
	r, e := inspect(context.Background(), Options{}, fakeHost(raw))
	if e != nil {
		t.Fatal(e)
	}
	f, ok := find(r, "BIND_MISSING")
	if !ok || f.Level != "warning" {
		t.Fatalf("%+v", r)
	}
}
func TestRemoteContextSkipsHostChecks(t *testing.T) {
	h := fakeHost(goodModel)
	h.Run = func(_ context.Context, command string, args ...string) ([]byte, error) {
		joined := command + " " + strings.Join(args, " ")
		if strings.HasPrefix(joined, "docker compose ") {
			return []byte(goodModel), nil
		}
		if strings.HasPrefix(joined, "docker context inspect") {
			return []byte("tcp://remote.example:2376"), nil
		}
		t.Fatalf("unexpected local operation: %s", joined)
		return nil, nil
	}
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil {
		t.Fatal(e)
	}
	if r.Summary.Blockers != 0 || !hasID(r, "CONTEXT") || hasID(r, "BIND_MISSING") {
		t.Fatalf("remote must skip local findings: %+v", r)
	}
}
func TestUnknownSocketStateDoesNotClaimFreePorts(t *testing.T) {
	h := fakeHost(goodModel)
	old := h.Run
	h.Run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "ss" {
			return nil, errors.New("ss unavailable")
		}
		return old(ctx, name, args...)
	}
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil {
		t.Fatal(e)
	}
	if hasID(r, "PORTS") || !hasID(r, "SOCKETS") {
		t.Fatalf("must not claim clear ports: %+v", r)
	}
}
func TestDiskWarningDoesNotClaimImageSize(t *testing.T) {
	h := fakeHost(`{"name":"x","services":{"api":{}}}`)
	h.Free = func(string) (uint64, error) { return 1024, nil }
	r, e := inspect(context.Background(), Options{MinFreeGiB: 2}, h)
	if e != nil {
		t.Fatal(e)
	}
	f, ok := find(r, "DISK_LOW")
	if !ok || !strings.Contains(f.Advice, "does not estimate") {
		t.Fatalf("%+v", r)
	}
}
func TestComposeErrorDoesNotLeakStderr(t *testing.T) {
	h := fakeHost(goodModel)
	h.Run = func(context.Context, string, ...string) ([]byte, error) {
		return nil, fmt.Errorf("SECRET_API_KEY=do-not-print")
	}
	r, e := inspect(context.Background(), Options{}, h)
	if e == nil {
		t.Fatal("must fail")
	}
	b := fmt.Sprint(r, e)
	if strings.Contains(b, "do-not-print") {
		t.Fatal("command error leaked")
	}
}
func TestSocketAddressOverlaps(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"127.0.0.1", "127.0.0.1", true},
		{"127.0.0.1", "0.0.0.0", true},
		{"0.0.0.0", "127.0.0.1", true},
		{"127.0.0.1", "192.0.2.1", false},
		{"::1", "::", true},
	}
	for _, c := range cases {
		if got := overlapping(c.a, c.b); got != c.want {
			t.Errorf("%s %s: %v", c.a, c.b, got)
		}
	}
}
func TestCLIArgHandling(t *testing.T) {
	var o, e bytes.Buffer
	if code := start([]string{"version"}, &o, &e); code != 0 || !strings.Contains(o.String(), "0.4.0-preview.1") {
		t.Fatal(o.String(), e.String(), code)
	}
	o.Reset()
	e.Reset()
	if code := start([]string{"check", "--format", "xml"}, &o, &e); code != 2 {
		t.Fatal(code)
	}
}
func TestComposeArgForwarding(t *testing.T) {
	o := Options{Files: []string{"first.yml", "second.yml"}, EnvFiles: []string{"custom.env"}, Profiles: []string{"debug"}}
	s := strings.Join(o.args(), " ")
	for _, want := range []string{"-f first.yml -f second.yml", "--env-file custom.env", "--profile debug", "config --format json"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s in %s", want, s)
		}
	}
}
func TestRealComposeDoesNotMutateFixtures(t *testing.T) {
	if os.Getenv("BEFOREUP_INTEGRATION") != "1" {
		t.Skip("set BEFOREUP_INTEGRATION=1 for local Docker Compose integration")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "compose.yaml")
	yaml := "services:\n  web:\n    image: nginx:alpine\n    volumes:\n      - type: bind\n        source: " + filepath.Join(dir, "uncreated") + "\n        target: /data\n        bind:\n          create_host_path: false\n"
	if e := os.WriteFile(file, []byte(yaml), 0600); e != nil {
		t.Fatal(e)
	}
	opt := Options{Files: []string{file}}
	ctx := context.Background()
	modelRaw, e := run(ctx, "docker", opt.args()...)
	if e != nil {
		t.Fatalf("docker compose config unavailable: %v", e)
	}
	if !strings.Contains(string(modelRaw), "create_host_path") {
		t.Fatal("docker did not preserve bind semantics")
	}
	if _, e := os.Stat(filepath.Join(dir, "uncreated")); !os.IsNotExist(e) {
		t.Fatal("config unexpectedly created bind directory")
	}
}

func TestInactiveComposeProfileIsIgnored(t *testing.T) {
	raw := `{"name":"x","services":{"debug":{"profiles":["debug"],"ports":[{"published":"8080","protocol":"tcp"}],"volumes":[{"type":"bind","source":"/missing/debug","bind":{"create_host_path":false}}]}}}`
	h := fakeHost(raw)
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil {
		t.Fatal(e)
	}
	if hasID(r, "BIND_MISSING") || hasID(r, "PORT_IN_USE") {
		t.Fatalf("inactive service triggered findings: %+v", r)
	}
	r, e = inspect(context.Background(), Options{Profiles: []string{"debug"}}, h)
	if e != nil {
		t.Fatal(e)
	}
	if !hasID(r, "BIND_MISSING") || !hasID(r, "PORT_IN_USE") {
		t.Fatalf("active profile missing findings: %+v", r)
	}
	h.Getenv = func(k string) string {
		if k == "COMPOSE_PROFILES" {
			return "debug,other"
		}
		return ""
	}
	r, e = inspect(context.Background(), Options{}, h)
	if e != nil || !hasID(r, "BIND_MISSING") {
		t.Fatalf("env profile ignored: %+v / %v", r, e)
	}
}
func TestDockerContextOverridesHostEnvironment(t *testing.T) {
	h := fakeHost(goodModel)
	h.Getenv = func(k string) string {
		switch k {
		case "DOCKER_CONTEXT":
			return "remote-prod"
		case "DOCKER_HOST":
			return "unix:///var/run/docker.sock"
		}
		return ""
	}
	h.Run = func(_ context.Context, command string, args ...string) ([]byte, error) {
		k := command + " " + strings.Join(args, " ")
		if strings.HasPrefix(k, "docker compose") {
			return []byte(goodModel), nil
		}
		if strings.Contains(k, "context inspect remote-prod") {
			return []byte("tcp://remote:2376"), nil
		}
		return nil, fmt.Errorf("unexpected call: %s", k)
	}
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil || !hasID(r, "CONTEXT") || hasID(r, "BIND_MISSING") {
		t.Fatalf("wrong endpoint used: %+v / %v", r, e)
	}
}
func TestBoundedOutput(t *testing.T) {
	b := &boundedOutput{}
	if n, e := b.Write([]byte("ok")); n != 2 || e != nil {
		t.Fatalf("%d %v", n, e)
	}
	large := make([]byte, maxCommandOutput)
	if _, e := b.Write(large); e == nil {
		t.Fatal("large stdout must be refused")
	}
	if len(b.data) != 2 {
		t.Fatal("buffer must not grow on refused output")
	}
}
func TestPortProtocolIsolation(t *testing.T) {
	list := parseSockets([]byte("LISTEN 0 128 127.0.0.1:8080 0.0.0.0:*\nUNCONN 0 0 [::]:5353 [::]:*\n"))
	if !occupied(8080, "0.0.0.0", list) || !occupied(5353, "::1", list) || occupied(9090, "127.0.0.1", list) {
		t.Fatalf("incorrect parse: %+v", list)
	}
}
func TestMalformedConfigIsNotReportedAsSuccess(t *testing.T) {
	h := fakeHost("{}")
	r, e := inspect(context.Background(), Options{}, h)
	if e == nil || r.Summary.Blockers == 0 || hasID(r, "PORTS") {
		t.Fatalf("invalid model: %+v / %v", r, e)
	}
}

func TestTwoServicesClaimSameHostPort(t *testing.T) {
	raw := `{"name":"x","services":{"one":{"ports":[{"host_ip":"0.0.0.0","published":"8088","protocol":"tcp"}]},"two":{"ports":[{"host_ip":"127.0.0.1","published":"8088","protocol":"tcp"}]}}}`
	r, e := inspect(context.Background(), Options{}, fakeHost(raw))
	if e != nil {
		t.Fatal(e)
	}
	f, found := find(r, "PORT_COLLISION")
	if !found || f.Level != "blocker" {
		t.Fatalf("duplicate host ports should block: %+v", r)
	}
}

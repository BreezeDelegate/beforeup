package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Finding struct {
	ID      string `json:"id"`
	Level   string `json:"level"`
	Service string `json:"service,omitempty"`
	Message string `json:"message"`
	Advice  string `json:"advice,omitempty"`
}
type Summary struct {
	Blockers int `json:"blockers"`
	Warnings int `json:"warnings"`
	Passed   int `json:"passed"`
}
type Report struct {
	Project  string    `json:"project,omitempty"`
	Findings []Finding `json:"findings"`
	Summary  Summary   `json:"summary"`
}

func (r *Report) add(id, level, service, message, advice string) {
	r.Findings = append(r.Findings, Finding{id, level, service, message, advice})
	switch level {
	case "blocker":
		r.Summary.Blockers++
	case "warning":
		r.Summary.Warnings++
	case "pass":
		r.Summary.Passed++
	}
}

type Options struct {
	Files, EnvFiles, Profiles []string
	MinFreeGiB                uint64
	Compare                   bool
}

func (o Options) args() []string {
	a := []string{"compose"}
	for _, v := range o.Files {
		a = append(a, "-f", v)
	}
	for _, v := range o.EnvFiles {
		a = append(a, "--env-file", v)
	}
	for _, v := range o.Profiles {
		a = append(a, "--profile", v)
	}
	return append(a, "config", "--format", "json")
}

type Host struct {
	Run    func(context.Context, string, ...string) ([]byte, error)
	Stat   func(string) (os.FileInfo, error)
	Free   func(string) (uint64, error)
	Getenv func(string) string
}

func realHost() Host { return Host{run, os.Stat, freeBytes, os.Getenv} }

const maxCommandOutput = 8 << 20

type boundedOutput struct{ data []byte }

func (b *boundedOutput) Write(p []byte) (int, error) {
	if len(p) > maxCommandOutput-len(b.data) {
		return 0, errors.New("command output exceeds limit")
	}
	b.data = append(b.data, p...)
	return len(p), nil
}
func run(ctx context.Context, command string, args ...string) ([]byte, error) {
	// Compose may include credentials in its resolved model. Never log command output.
	cmd := exec.CommandContext(ctx, command, args...)
	var output boundedOutput
	cmd.Stdout = &output
	cmd.Stderr = io.Discard
	err := cmd.Run()
	return output.data, err
}
func freeBytes(path string) (uint64, error) {
	var s syscall.Statfs_t
	if err := syscall.Statfs(path, &s); err != nil {
		return 0, err
	}
	return s.Bavail * uint64(s.Bsize), nil
}

type service struct {
	Profiles []string `json:"profiles"`
	Ports    []struct {
		Published string `json:"published"`
		HostIP    string `json:"host_ip"`
		Protocol  string `json:"protocol"`
	} `json:"ports"`
	Volumes []struct {
		Type   string `json:"type"`
		Source string `json:"source"`
		Target string `json:"target"`
		Bind   struct {
			CreateHostPath *bool `json:"create_host_path"`
		} `json:"bind"`
	} `json:"volumes"`
}
type model struct {
	Name     string             `json:"name"`
	Services map[string]service `json:"services"`
	Volumes  map[string]struct {
		Name string `json:"name"`
	} `json:"volumes"`
}

func localDocker(ctx context.Context, h Host) bool {
	// An explicit Docker context overrides DOCKER_HOST.
	if name := h.Getenv("DOCKER_CONTEXT"); name != "" {
		v, err := h.Run(ctx, "docker", "context", "inspect", name, "--format", "{{.Endpoints.docker.Host}}")
		return err == nil && strings.HasPrefix(strings.TrimSpace(string(v)), "unix://")
	}
	if v := h.Getenv("DOCKER_HOST"); v != "" {
		return strings.HasPrefix(v, "unix://")
	}
	v, err := h.Run(ctx, "docker", "context", "inspect", "--format", "{{.Endpoints.docker.Host}}")
	return err == nil && strings.HasPrefix(strings.TrimSpace(string(v)), "unix://")
}

type socket struct {
	ip   string
	port int
}

func parseSockets(raw []byte) []socket {
	var out []socket
	for _, line := range strings.Split(string(raw), "\n") {
		f := strings.Fields(line)
		if len(f) < 5 {
			continue
		}
		addr := f[3]
		p := strings.LastIndexByte(addr, ':')
		if p < 0 {
			continue
		}
		n, e := strconv.Atoi(addr[p+1:])
		if e != nil || n < 1 || n > 65535 {
			continue
		}
		out = append(out, socket{strings.Trim(addr[:p], "[]"), n})
	}
	return out
}
func overlapping(a, b string) bool {
	any := func(s string) bool { return s == "" || s == "*" || s == "0.0.0.0" || s == "::" }
	if any(a) || any(b) {
		return true
	}
	ipA, ipB := net.ParseIP(a), net.ParseIP(b)
	if ipA != nil && ipB != nil {
		return ipA.Equal(ipB)
	}
	return a == b
}
func occupied(port int, ip string, items []socket) bool {
	for _, s := range items {
		if s.port == port && overlapping(ip, s.ip) {
			return true
		}
	}
	return false
}
func inspect(parent context.Context, o Options, h Host) (Report, error) {
	r := Report{Findings: []Finding{}}
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()
	raw, e := h.Run(ctx, "docker", o.args()...)
	if e != nil {
		r.add("COMPOSE", "blocker", "", "Docker Compose could not resolve the configuration.", "Run docker compose config locally to see the detailed error.")
		return r, errors.New("compose validation failed")
	}
	if len(raw) > 8<<20 {
		r.add("COMPOSE", "blocker", "", "Resolved configuration exceeds the 8 MiB safety limit.", "Simplify the Compose model.")
		return r, errors.New("compose model too large")
	}
	var m model
	if json.Unmarshal(raw, &m) != nil || m.Name == "" {
		r.add("COMPOSE", "blocker", "", "Compose returned no usable services.", "Inspect docker compose config.")
		return r, errors.New("invalid compose output")
	}
	r.Project = m.Name
	r.add("COMPOSE", "pass", "", "Docker Compose configuration is valid.", "")
	if len(m.Services) == 0 {
		r.add("NO_ACTIVE_SERVICES", "warning", "", "Compose selected no active services.", "Enable the relevant profile or check the selected Compose files. There is nothing to start in this configuration.")
		return r, nil
	}
	if !localDocker(ctx, h) {
		r.add("CONTEXT", "warning", "", "Docker endpoint is remote or its location could not be confirmed.", "Local socket, filesystem and disk checks were skipped. Run BeforeUp on the Docker host.")
		return r, nil
	}
	root, daemonErr := h.Run(ctx, "docker", "info", "--format", "{{.DockerRootDir}}")
	if daemonErr != nil {
		r.add("DAEMON", "blocker", "", "Docker Engine cannot be reached from this account.", "Check docker info and your Docker socket permissions.")
	} else {
		r.add("DAEMON", "pass", "", "Docker Engine is reachable.", "")
	}
	tcpRaw, tcpErr := h.Run(ctx, "ss", "-H", "-ltn")
	udpRaw, udpErr := h.Run(ctx, "ss", "-H", "-lun")
	tcp, udp := parseSockets(tcpRaw), parseSockets(udpRaw)
	if tcpErr != nil || udpErr != nil {
		r.add("SOCKETS", "warning", "", "Could not read the host's listening sockets.", "Install iproute2 (ss) or check ports manually.")
	}
	services := make([]string, 0, len(m.Services))
	for name := range m.Services {
		services = append(services, name)
	}
	sort.Strings(services)
	seen := map[string]bool{}
	type claim struct {
		service string
		host    string
		port    int
		proto   string
	}
	var claims []claim
	for _, name := range services {
		svc := m.Services[name]
		if !profileEnabled(svc.Profiles, o.Profiles, h.Getenv("COMPOSE_PROFILES")) {
			continue
		}
		for _, p := range svc.Ports {
			if p.Published == "" {
				continue
			}
			n, e := strconv.Atoi(p.Published)
			if e != nil || n < 1 || n > 65535 {
				r.add("PORT_RANGE", "warning", name, "Published port is not a single numeric port.", "Review host port ranges manually.")
				continue
			}
			protocol := strings.ToLower(p.Protocol)
			if protocol == "" {
				protocol = "tcp"
			}
			for _, c := range claims {
				if c.port == n && c.proto == protocol && c.service != name && overlapping(c.host, p.HostIP) {
					r.add("PORT_COLLISION", "blocker", name, fmt.Sprintf("Two services publish overlapping %s port %d bindings (%s and %s).", protocol, n, c.service, name), "Change one published host port before starting the stack.")
				}
			}
			claims = append(claims, claim{name, p.HostIP, n, protocol})
			key := fmt.Sprintf("%s:%d/%s", p.HostIP, n, protocol)
			if seen[key] {
				continue
			}
			seen[key] = true
			var items []socket
			var err error
			switch protocol {
			case "tcp":
				items, err = tcp, tcpErr
			case "udp":
				items, err = udp, udpErr
			default:
				continue
			}
			attributed := false
			if daemonErr == nil {
				owners, ownerErr := portOwners(ctx, h, n, protocol, p.HostIP)
				if ownerErr == nil && len(owners) > 0 {
					own := true
					for _, owner := range owners {
						if owner.Project != m.Name || owner.Service != name || !owner.Exact {
							own = false
							break
						}
					}
					if own {
						r.add("PORT_OWNED_BY_SERVICE", "pass", name, fmt.Sprintf("Host %s port %d is already published by this running service.", protocol, n), "This is expected for a currently running stack.")
					} else if len(owners) == 1 && owners[0].Project == m.Name && owners[0].Service == name {
						r.add("PORT_BINDING_CHANGED", "warning", name, fmt.Sprintf("The running service publishes %s port %d on a different host address.", protocol, n), "The proposed binding differs from the running container; check local and remote access before deploying.")
					} else {
						o := owners[0]
						r.add("PORT_USED_BY_OTHER", "warning", name, fmt.Sprintf("Host %s port %d is published by Docker project %s, service %s.", protocol, n, o.Project, o.Service), "Check which container owns the binding before deploying; another running project may be affected.")
					}
					attributed = true
				}
			}
			if !attributed && err == nil && occupied(n, p.HostIP, items) {
				r.add("PORT_IN_USE", "warning", name, fmt.Sprintf("Host %s port %d already has a listener.", protocol, n), "Check the listener with ss. The owner could not be confirmed as this Compose service.")
			}
		}
		for _, v := range svc.Volumes {
			if v.Type != "bind" || v.Source == "" {
				continue
			}
			source := filepath.Clean(v.Source)
			_, err := h.Stat(source)
			if err == nil {
				continue
			}
			if os.IsNotExist(err) {
				level, advice := "warning", "Docker may create an empty directory for a short bind mount. Check that this is intended."
				if v.Bind.CreateHostPath != nil && !*v.Bind.CreateHostPath {
					level, advice = "blocker", "Create the required path or change the bind declaration before startup."
				}
				r.add("BIND_MISSING", level, name, "A bind source for this service does not exist.", advice)
			} else {
				r.add("BIND_UNKNOWN", "warning", name, "A bind source for this service could not be inspected.", "Check permissions and host mounts.")
			}
		}
	}
	if tcpErr == nil && udpErr == nil && !hasID(r, "PORT_IN_USE") && !hasID(r, "PORT_USED_BY_OTHER") && !hasID(r, "PORT_COLLISION") && !hasID(r, "PORT_BINDING_CHANGED") {
		r.add("PORTS", "pass", "", "No unexplained published-port conflicts detected.", "")
	}
	if o.MinFreeGiB > 0 && daemonErr == nil {
		path := strings.TrimSpace(string(root))
		if path == "" {
			path = "/"
		}
		free, e := h.Free(path)
		if e != nil {
			r.add("DISK_UNKNOWN", "warning", "", "Cannot inspect available space in Docker's data filesystem.", "Check free space on the Docker host.")
		} else if free/1073741824 < o.MinFreeGiB {
			r.add("DISK_LOW", "warning", "", fmt.Sprintf("Docker's filesystem has about %d GiB free; threshold is %d GiB.", free/1073741824, o.MinFreeGiB), "This warns about limited headroom; it does not estimate image download size.")
		} else {
			r.add("DISK", "pass", "", fmt.Sprintf("At least %d GiB free in Docker's data filesystem.", o.MinFreeGiB), "")
		}
	}
	if o.Compare {
		if daemonErr != nil {
			r.add("BASELINE_UNKNOWN", "warning", "", "Running service comparison needs Docker Engine access.", "Fix Docker access and rerun compare.")
		} else {
			comparison(ctx, h, m, o.Profiles, h.Getenv("COMPOSE_PROFILES"), &r)
		}
	}
	return r, nil
}
func hasID(r Report, id string) bool {
	for _, f := range r.Findings {
		if f.ID == id {
			return true
		}
	}
	return false
}

// A service behind a Compose profile is not started unless that profile is enabled.
func profileEnabled(serviceProfiles, selected []string, fromEnv string) bool {
	if len(serviceProfiles) == 0 {
		return true
	}
	active := make(map[string]bool, len(selected))
	for _, p := range selected {
		if p == "*" {
			return true
		}
		active[p] = true
	}
	for _, p := range strings.Split(fromEnv, ",") {
		if p = strings.TrimSpace(p); p != "" {
			if p == "*" {
				return true
			}
			active[p] = true
		}
	}
	for _, p := range serviceProfiles {
		if active[p] {
			return true
		}
	}
	return false
}

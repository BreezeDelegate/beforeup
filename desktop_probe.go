package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Windows checks are limited to a local Docker Desktop Linux Engine.
// Its host ports, bind sources and disk live in a VM, not on Windows.
func inspectWindowsDesktop(ctx context.Context, o Options, h Host, m model, r *Report) {
	if !localDockerEndpoint(ctx, h, "npipe://") {
		r.add("CONTEXT", "warning", "", "Docker endpoint is remote or its local Windows named pipe could not be confirmed.", "Only Compose configuration was resolved; Docker host checks and running-state comparison were skipped.")
		return
	}
	osType, err := h.Run(ctx, "docker", "info", "--format", "{{.OSType}}")
	if err != nil {
		r.add("DAEMON", "blocker", "", "Docker Engine cannot be reached from this account.", "Start Docker Desktop and verify Docker access.")
		if o.Compare {
			r.add("BASELINE_UNKNOWN", "warning", "", "Running service comparison needs Docker Engine access.", "No changes were confirmed without a running baseline.")
		}
		return
	}
	if strings.TrimSpace(strings.ToLower(string(osType))) != "linux" {
		r.add("WINDOWS_ENGINE_UNSUPPORTED", "warning", "", "The selected Docker Engine is not a Linux container engine.", "BeforeUp has not validated Windows-container comparisons; switch to Docker Desktop Linux containers for a limited read-only check.")
		return
	}
	name, err := h.Run(ctx, "docker", "info", "--format", "{{.OperatingSystem}}")
	if err != nil || !strings.Contains(strings.ToLower(string(name)), "docker desktop") {
		r.add("CONTEXT", "warning", "", "The Docker Engine was not independently identified as Docker Desktop.", "Running-state and workstation host checks were skipped rather than assuming a local Linux VM.")
		return
	}
	r.add("DAEMON", "pass", "", "Docker Desktop Linux Engine is reachable.", "")
	r.add("DESKTOP_VM", "warning", "", "Docker Desktop uses a Linux VM; Windows host port, bind path and disk checks were skipped.", "Inspect host-side conditions in the Docker VM separately. A clean comparison is not a deployment guarantee.")
	inspectWindowsDeclaredPorts(o, h, m, r)
	if o.Compare {
		comparison(ctx, h, m, o.Profiles, h.Getenv("COMPOSE_PROFILES"), r)
	}
}

func inspectWindowsDeclaredPorts(o Options, h Host, m model, r *Report) {
	services := make([]string, 0, len(m.Services))
	for name := range m.Services {
		services = append(services, name)
	}
	sort.Strings(services)
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
			n, err := strconv.Atoi(p.Published)
			if err != nil || n < 1 || n > 65535 {
				r.add("PORT_RANGE", "warning", name, "Published port is not a single numeric port.", "Review host port ranges manually.")
				continue
			}
			protocol := strings.ToLower(p.Protocol)
			if protocol == "" {
				protocol = "tcp"
			}
			for _, previous := range claims {
				if previous.port != n || previous.proto != protocol || previous.service == name || !overlapping(previous.host, p.HostIP) {
					continue
				}
				if uncertainPortOverlap(previous.host, p.HostIP) {
					r.add("PORT_DUAL_STACK_UNCERTAIN", "warning", name, fmt.Sprintf("Two services publish %s port %d with potentially overlapping IPv4/IPv6 bindings.", protocol, n), "Check IPv6-only socket mode and Docker userland-proxy behavior before deploying.")
				} else {
					r.add("PORT_COLLISION", "blocker", name, fmt.Sprintf("Two services publish overlapping %s port %d bindings (%s and %s).", protocol, n, previous.service, name), "Change one published host port before starting the stack.")
				}
			}
			claims = append(claims, claim{name, p.HostIP, n, protocol})
		}
	}
}

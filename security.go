package main

import (
	"path/filepath"
	"sort"
	"strings"
)

// securityWarnings inspects only the resolved Compose model. It does not query
// the host or treat a risky declaration as proof of an exploitable vulnerability.
func securityWarnings(m model, selectedProfiles []string, envProfiles string, r *Report) {
	names := make([]string, 0, len(m.Services))
	for name := range m.Services {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		svc := m.Services[name]
		if !profileEnabled(svc.Profiles, selectedProfiles, envProfiles) {
			continue
		}
		if svc.Privileged {
			r.add("SECURITY_PRIVILEGED", "warning", name,
				"Service requests privileged container mode.",
				"Remove privileged: true and grant only the capabilities or devices actually needed.")
		}
		if hasElevatedCapability(svc.CapAdd) {
			r.add("SECURITY_CAPABILITIES", "warning", name,
				"Service adds a sensitive Linux capability.",
				"Review cap_add and avoid broad capabilities such as ALL or SYS_ADMIN unless essential.")
		}
		if disablesConfinement(svc.SecurityOpt) {
			r.add("SECURITY_CONFINEMENT", "warning", name,
				"Service disables a container confinement control.",
				"Review security_opt and retain seccomp, AppArmor or SELinux restrictions where possible.")
		}
		socketMounted := false
		for _, mount := range svc.Volumes {
			if mount.Type == "bind" && strings.EqualFold(filepath.Base(filepath.Clean(mount.Source)), "docker.sock") {
				socketMounted = true
				break
			}
		}
		if socketMounted {
			r.add("SECURITY_DOCKER_SOCKET", "warning", name,
				"Service bind-mounts a path named docker.sock; Docker API access may give control of the host.",
				"A read-only bind is not a read-only Docker API. Avoid mounting the Docker socket or restrict API access.")
		}
		if strings.EqualFold(svc.NetworkMode, "host") {
			r.add("SECURITY_HOST_NETWORK", "warning", name,
				"Service shares the host network namespace.",
				"Prefer a dedicated container network unless host networking is required.")
		}
		if strings.EqualFold(svc.PID, "host") {
			r.add("SECURITY_HOST_PID", "warning", name,
				"Service shares the host PID namespace.",
				"Use an isolated PID namespace unless access to host processes is required.")
		}
		if strings.EqualFold(svc.IPC, "host") {
			r.add("SECURITY_HOST_IPC", "warning", name,
				"Service shares the host IPC namespace.",
				"Use an isolated IPC namespace unless host shared memory access is required.")
		}
	}
}

func hasElevatedCapability(capabilities []string) bool {
	for _, capability := range capabilities {
		c := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(capability)), "CAP_")
		switch c {
		case "ALL", "SYS_ADMIN", "SYS_MODULE", "SYS_PTRACE", "SYS_RAWIO",
			"NET_ADMIN", "DAC_READ_SEARCH", "DAC_OVERRIDE", "BPF", "PERFMON":
			return true
		}
	}
	return false
}

func disablesConfinement(options []string) bool {
	for _, option := range options {
		switch strings.ToLower(strings.TrimSpace(option)) {
		case "seccomp=unconfined", "seccomp:unconfined",
			"apparmor=unconfined", "apparmor:unconfined",
			"label=disable", "label:disable":
			return true
		}
	}
	return false
}

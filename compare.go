package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type hostBinding struct {
	HostIP   string `json:"HostIp"`
	HostPort string `json:"HostPort"`
}
type containerMount struct {
	Type        string `json:"Type"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	Name        string `json:"Name"`
}
type runningContainer struct {
	Project string
	Service string
	ID      string
	Ports   map[string][]hostBinding
	Mounts  []containerMount
}

func checkedID(s string) bool {
	if len(s) < 12 || len(s) > 64 {
		return false
	}
	for _, r := range s {
		if r >= '0' && r <= '9' || r >= 'a' && r <= 'f' {
			continue
		}
		return false
	}
	return true
}
func safeLabel(label string) string {
	if len(label) == 0 || len(label) > 80 {
		return "unidentified"
	}
	for _, r := range label {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' {
			continue
		}
		return "unidentified"
	}
	return label
}
func readContainer(ctx context.Context, h Host, id string) (runningContainer, error) {
	if !checkedID(id) {
		return runningContainer{}, errors.New("invalid container ID")
	}
	raw, err := h.Run(ctx, "docker", "inspect", "--format", "{{json .HostConfig.PortBindings}}|||{{json .Mounts}}", id)
	if err != nil {
		return runningContainer{}, errors.New("container inspect unavailable")
	}
	parts := strings.SplitN(strings.TrimSpace(string(raw)), "|||", 2)
	if len(parts) != 2 {
		return runningContainer{}, errors.New("unexpected inspect output")
	}
	c := runningContainer{Ports: map[string][]hostBinding{}}
	if parts[0] != "null" && json.Unmarshal([]byte(parts[0]), &c.Ports) != nil {
		return runningContainer{}, errors.New("invalid port bindings")
	}
	if parts[1] != "null" && json.Unmarshal([]byte(parts[1]), &c.Mounts) != nil {
		return runningContainer{}, errors.New("invalid mount metadata")
	}
	return c, nil
}

// Runtime environment values stay in memory. Never include them in findings.
func readRuntimeMetadata(ctx context.Context, h Host, id string) (string, map[string]string, error) {
	if !checkedID(id) {
		return "", nil, errors.New("invalid container ID")
	}
	raw, err := h.Run(ctx, "docker", "inspect", "--format", "{{json .Config.Image}}|||{{json .Config.Env}}", id)
	if err != nil {
		return "", nil, errors.New("runtime metadata unavailable")
	}
	parts := strings.SplitN(strings.TrimSpace(string(raw)), "|||", 2)
	if len(parts) != 2 {
		return "", nil, errors.New("unexpected runtime metadata")
	}
	var image string
	if err := json.Unmarshal([]byte(parts[0]), &image); err != nil {
		return "", nil, errors.New("invalid image metadata")
	}
	var entries []string
	if err := json.Unmarshal([]byte(parts[1]), &entries); err != nil {
		return "", nil, errors.New("invalid environment metadata")
	}
	env := make(map[string]string, len(entries))
	for _, entry := range entries {
		pair := strings.SplitN(entry, "=", 2)
		if len(pair) == 2 {
			env[pair[0]] = pair[1]
		}
	}
	return image, env, nil
}
func parseOwners(raw []byte) ([]struct{ id, project, service string }, error) {
	var out []struct{ id, project, service string }
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return out, nil
	}
	if len(lines) > 32 {
		return nil, errors.New("too many published-port candidates")
	}
	for _, line := range lines {
		p := strings.Split(line, "|")
		if len(p) != 3 || !checkedID(p[0]) {
			return nil, errors.New("invalid Docker container listing")
		}
		out = append(out, struct{ id, project, service string }{p[0], safeLabel(p[1]), safeLabel(p[2])})
	}
	return out, nil
}
func matchingPort(ports map[string][]hostBinding, number int, proto, ip string, exact bool) bool {
	for descriptor, bindings := range ports {
		name := strings.SplitN(descriptor, "/", 2)
		if len(name) != 2 || name[1] != proto {
			continue
		}
		for _, b := range bindings {
			n, e := strconv.Atoi(b.HostPort)
			if e != nil || n != number {
				continue
			}
			if exact {
				if normalizeHostIP(ip) == normalizeHostIP(b.HostIP) {
					return true
				}
			} else if overlapping(ip, b.HostIP) {
				return true
			}
		}
	}
	return false
}
func normalizeHostIP(s string) string {
	if s == "" || s == "0.0.0.0" {
		return "0.0.0.0"
	}
	if s == "::" {
		return "::"
	}
	return s
}

type owner struct {
	Project, Service string
	Exact            bool
}

func portOwners(ctx context.Context, h Host, port int, protocol, ip string) ([]owner, error) {
	raw, err := h.Run(ctx, "docker", "ps", "--filter", fmt.Sprintf("publish=%d", port), "--format", `{{.ID}}|{{.Label "com.docker.compose.project"}}|{{.Label "com.docker.compose.service"}}`)
	if err != nil {
		return nil, err
	}
	entries, err := parseOwners(raw)
	if err != nil {
		return nil, err
	}
	var owners []owner
	for _, o := range entries {
		c, err := readContainer(ctx, h, o.id)
		if err != nil {
			return nil, err
		}
		if matchingPort(c.Ports, port, protocol, ip, false) {
			owners = append(owners, owner{Project: o.project, Service: o.service, Exact: matchingPort(c.Ports, port, protocol, ip, true)})
		}
	}
	return owners, nil
}
func projectContainers(ctx context.Context, h Host, project string) (map[string]runningContainer, error) {
	raw, err := h.Run(ctx, "docker", "ps", "--filter", "label=com.docker.compose.project="+project, "--format", `{{.ID}}|{{.Label "com.docker.compose.service"}}`)
	if err != nil {
		return nil, err
	}
	current := map[string]runningContainer{}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return current, nil
	}
	if len(lines) > 64 {
		return nil, errors.New("too many containers for a safe comparison")
	}
	for _, line := range lines {
		p := strings.Split(line, "|")
		if len(p) != 2 || !checkedID(p[0]) {
			return nil, errors.New("invalid running container listing")
		}
		service := safeLabel(p[1])
		c, err := readContainer(ctx, h, p[0])
		if err != nil {
			return nil, err
		}
		c.Service = service
		c.Project = project
		c.ID = p[0]
		if _, duplicate := current[service]; duplicate {
			return nil, errors.New("multiple running containers for the same service are not yet supported")
		}
		current[service] = c
	}
	return current, nil
}
func plannedPublished(s service) map[string]bool {
	out := map[string]bool{}
	for _, p := range s.Ports {
		if p.Published == "" {
			continue
		}
		proto := strings.ToLower(p.Protocol)
		if proto == "" {
			proto = "tcp"
		}
		out[normalizeHostIP(p.HostIP)+":"+p.Published+"/"+proto] = true
	}
	return out
}
func runningPublished(c runningContainer) map[string]bool {
	out := map[string]bool{}
	for key, entries := range c.Ports {
		protoParts := strings.SplitN(key, "/", 2)
		if len(protoParts) != 2 {
			continue
		}
		for _, p := range entries {
			if p.HostPort != "" {
				out[normalizeHostIP(p.HostIP)+":"+p.HostPort+"/"+protoParts[1]] = true
			}
		}
	}
	return out
}
func comparison(ctx context.Context, h Host, model model, selected []string, envProfiles string, r *Report) {
	live, err := projectContainers(ctx, h, model.Name)
	if err != nil {
		r.add("BASELINE_UNKNOWN", "warning", "", "Running Compose service state could not be inspected.", "Check Docker access and try again; no comparison is claimed.")
		return
	}
	if len(live) == 0 {
		r.add("NO_RUNNING_STACK", "warning", "", "No running containers were found for this Compose project.", "Only host preflight checks are available before the first deployment.")
		return
	}
	names := make([]string, 0, len(model.Services))
	for name := range model.Services {
		names = append(names, name)
	}
	sort.Strings(names)
	compared := 0
	for _, name := range names {
		planned := model.Services[name]
		if !profileEnabled(planned.Profiles, selected, envProfiles) {
			continue
		}
		existing, ok := live[name]
		if !ok {
			r.add("SERVICE_NEW", "warning", name, "No running container was found for this service.", "Review changes carefully; there is no running service baseline.")
			continue
		}
		compared++
		if planned.Image != "" || len(planned.Environment) > 0 {
			image, currentEnv, metaErr := readRuntimeMetadata(ctx, h, existing.ID)
			if metaErr != nil {
				r.add("RUNTIME_METADATA_UNKNOWN", "warning", name, "The running image and environment could not be compared.", "Check Docker access; do not assume the application configuration is unchanged.")
			} else {
				if planned.Image != "" && planned.Image != image {
					r.add("IMAGE_REFERENCE_CHANGED", "warning", name, "The proposed image reference differs from the running container.", "Verify the image and plan for potential container recreation; this does not check image digests.")
				}
				changes := 0
				for key, want := range planned.Environment {
					if want == nil {
						continue
					}
					got, exists := currentEnv[key]
					if !exists || got != *want {
						changes++
					}
				}
				if changes > 0 {
					r.add("ENV_CHANGED", "warning", name, fmt.Sprintf("%d declared environment setting(s) differ from the running container.", changes), "docker compose restart does not apply new environment values. Review the proposed settings and use docker compose up -d when appropriate.")
				}
			}
		}
		desiredPorts := plannedPublished(planned)
		oldPorts := runningPublished(existing)
		for key := range desiredPorts {
			if !oldPorts[key] {
				r.add("PORT_ADDED", "warning", name, "Proposed published port is not used by the running service: "+key, "Check existing listeners and update firewall or proxy configuration if necessary.")
			}
		}
		for key := range oldPorts {
			if !desiredPorts[key] {
				r.add("PORT_REMOVED", "warning", name, "Running service publishes a port absent from the proposal: "+key, "Check clients, reverse proxies and firewall rules before applying the change.")
			}
		}
		oldMounts := map[string]containerMount{}
		for _, v := range existing.Mounts {
			if v.Type == "bind" || v.Type == "volume" {
				oldMounts[v.Destination] = v
			}
		}
		plannedMounts := map[string]struct{ Kind, Source string }{}
		for _, v := range planned.Volumes {
			if v.Type == "bind" || v.Type == "volume" {
				source := v.Source
				if v.Type == "volume" {
					if vol, ok := model.Volumes[v.Source]; ok && vol.Name != "" {
						source = vol.Name
					}
				}
				plannedMounts[v.Target] = struct{ Kind, Source string }{v.Type, source}
			}
		}
		for target, prev := range oldMounts {
			want, ok := plannedMounts[target]
			if !ok {
				r.add("MOUNT_REMOVED", "warning", name, "A running data mount is absent from the proposed service at "+target+".", "The existing data may remain on disk, but the new container may no longer see it. Check the migration plan.")
				continue
			}
			if want.Kind != prev.Type || (prev.Type == "bind" && want.Source != prev.Source) || (prev.Type == "volume" && prev.Name != "" && want.Source != "" && want.Source != prev.Name) {
				r.add("MOUNT_SOURCE_CHANGED", "warning", name, "The proposed data source changes at "+target+".", "Check where existing data lives and verify a safe migration before redeploying.")
			}
		}
	}
	for serviceName := range live {
		if _, exists := model.Services[serviceName]; !exists {
			r.add("SERVICE_ABSENT_FROM_PROPOSAL", "warning", serviceName, "A running service is absent from the proposed Compose model.", "It may remain as an orphan unless you explicitly remove it. Review the deployment plan without deleting containers blindly.")
		}
	}
	if compared > 0 && !hasID(*r, "ENV_CHANGED") && !hasID(*r, "IMAGE_REFERENCE_CHANGED") && !hasID(*r, "RUNTIME_METADATA_UNKNOWN") && !hasID(*r, "SERVICE_ABSENT_FROM_PROPOSAL") && !hasID(*r, "PORT_ADDED") && !hasID(*r, "PORT_REMOVED") && !hasID(*r, "MOUNT_REMOVED") && !hasID(*r, "MOUNT_SOURCE_CHANGED") && !hasID(*r, "SERVICE_NEW") {
		r.add("RUNNING_MATCH", "pass", "", "No changes were detected in the inspected ports, mounts, declared environment or image references.", "Image digests, application health and other runtime settings are not analyzed.")
	}
}

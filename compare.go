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

// A locally updated tag can point to another image than a running container.
// This does not query the registry or pull an image. Unknown results stay unknown.
func localImageUpdated(ctx context.Context, h Host, containerID, reference string) bool {
	if !checkedID(containerID) || reference == "" {
		return false
	}
	running, err := h.Run(ctx, "docker", "inspect", "--format", "{{.Image}}", containerID)
	if err != nil {
		return false
	}
	local, err := h.Run(ctx, "docker", "image", "inspect", "--format", "{{.Id}}", reference)
	if err != nil {
		return false
	}
	a, b := strings.TrimSpace(string(running)), strings.TrimSpace(string(local))
	return strings.HasPrefix(a, "sha256:") && strings.HasPrefix(b, "sha256:") && len(a) == 71 && len(b) == 71 && a != b
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
func projectContainers(ctx context.Context, h Host, project string) (map[string]runningContainer, map[string]bool, error) {
	raw, err := h.Run(ctx, "docker", "ps", "--filter", "label=com.docker.compose.project="+project, "--format", `{{.ID}}|{{.Label "com.docker.compose.service"}}`)
	if err != nil {
		return nil, nil, err
	}
	current := map[string]runningContainer{}
	scaled := map[string]bool{}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return current, scaled, nil
	}
	if len(lines) > 64 {
		return nil, nil, errors.New("too many containers for a safe comparison")
	}
	ids := map[string]string{}
	for _, line := range lines {
		fields := strings.Split(line, "|")
		if len(fields) != 2 || !checkedID(fields[0]) {
			return nil, nil, errors.New("invalid running container listing")
		}
		service := safeLabel(fields[1])
		if _, duplicate := ids[service]; duplicate {
			scaled[service] = true
			continue
		}
		ids[service] = fields[0]
	}
	// Only inspect services with a single running container. This avoids both
	// needless Docker work and opaque whole-stack failures on scaled services.
	for service, id := range ids {
		if scaled[service] {
			continue
		}
		c, err := readContainer(ctx, h, id)
		if err != nil {
			return nil, nil, err
		}
		c.Service = service
		c.Project = project
		c.ID = id
		current[service] = c
	}
	return current, scaled, nil
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
	live, scaled, err := projectContainers(ctx, h, model.Name)
	if err != nil {
		r.add("BASELINE_UNKNOWN", "warning", "", "Running Compose service state could not be inspected.", "Check Docker access and try again; no comparison is claimed.")
		return
	}
	if len(live) == 0 && len(scaled) == 0 {
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
		if scaled[name] {
			r.add("REPLICAS_UNSUPPORTED", "warning", name, "Multiple running containers exist for this service; a reliable comparison is not available.", "Inspect each replica with Docker before changing this service. Other services are still checked.")
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
				} else if planned.Image != "" && planned.Platform == "" && localImageUpdated(ctx, h, existing.ID, planned.Image) {
					r.add("LOCAL_IMAGE_UPDATED", "warning", name, "The local image for this tag differs from the image used by the running container.", "A newer image is present locally. Review the deployment before recreating the container; this check does not query the registry.")
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
		removedMounts, changedMounts := 0, 0
		for target, prev := range oldMounts {
			want, ok := plannedMounts[target]
			if !ok {
				removedMounts++
				continue
			}
			if want.Kind != prev.Type || (prev.Type == "bind" && want.Source != prev.Source) || (prev.Type == "volume" && prev.Name != "" && want.Source != "" && want.Source != prev.Name) {
				changedMounts++
			}
		}
		if removedMounts > 0 {
			r.add("MOUNT_REMOVED", "warning", name, fmt.Sprintf("%d running data mount(s) are absent from the proposed service.", removedMounts), "Existing data may remain on disk, but the new container may no longer see it. Inspect mount targets locally and check the migration plan.")
		}
		if changedMounts > 0 {
			r.add("MOUNT_SOURCE_CHANGED", "warning", name, fmt.Sprintf("%d proposed data mount source(s) differ from the running service.", changedMounts), "Check mount targets and sources locally and verify a safe migration before redeploying.")
		}
	}
	orphanNames := make([]string, 0)
	for serviceName := range live {
		if _, exists := model.Services[serviceName]; !exists {
			orphanNames = append(orphanNames, serviceName)
		}
	}
	for serviceName := range scaled {
		if _, exists := model.Services[serviceName]; !exists {
			orphanNames = append(orphanNames, serviceName)
		}
	}
	sort.Strings(orphanNames)
	for _, serviceName := range orphanNames {
		if scaled[serviceName] {
			r.add("SERVICE_ABSENT_FROM_PROPOSAL", "warning", serviceName, "Multiple running containers belong to a service absent from the proposed Compose model.", "These replicas may remain as orphans unless intentionally removed. Review each container before changing the stack.")
		} else {
			r.add("SERVICE_ABSENT_FROM_PROPOSAL", "warning", serviceName, "A running service is absent from the proposed Compose model.", "It may remain as an orphan unless you explicitly remove it. Review the deployment plan without deleting containers blindly.")
		}
	}
	if compared > 0 && len(scaled) == 0 && !hasID(*r, "ENV_CHANGED") && !hasID(*r, "IMAGE_REFERENCE_CHANGED") && !hasID(*r, "LOCAL_IMAGE_UPDATED") && !hasID(*r, "RUNTIME_METADATA_UNKNOWN") && !hasID(*r, "SERVICE_ABSENT_FROM_PROPOSAL") && !hasID(*r, "PORT_ADDED") && !hasID(*r, "PORT_REMOVED") && !hasID(*r, "MOUNT_REMOVED") && !hasID(*r, "MOUNT_SOURCE_CHANGED") && !hasID(*r, "SERVICE_NEW") {
		r.add("RUNNING_MATCH", "pass", "", "No changes were detected in the inspected ports, mounts, declared environment or image references.", "Image digests, application health and other runtime settings are not analyzed.")
	}
}

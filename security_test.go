package main

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestSecurityWarningsCategoriesAndRedaction(t *testing.T) {
	const resolved = `{
		"name": "security-test",
		"services": {
			"risky": {
				"privileged": true,
				"cap_add": ["CAP_SYS_ADMIN", "NET_ADMIN"],
				"security_opt": ["seccomp=unconfined", "apparmor=unconfined"],
				"network_mode": "host",
				"pid": "host",
				"ipc": "host",
				"environment": {"INTERNAL_TOKEN": "SECRET_VALUE_NEVER_PRINT"},
				"volumes": [
					{"type": "bind", "source": "/secret/SECRET_BIND_PATH/docker.sock", "target": "/run/docker.sock", "read_only": true},
					{"type": "bind", "source": "/run/user/1000/docker.sock", "target": "/another/socket"}
				]
			}
		}
	}`
	var m model
	if err := json.Unmarshal([]byte(resolved), &m); err != nil {
		t.Fatal(err)
	}
	r := Report{}
	securityWarnings(m, nil, "", &r)
	want := []string{
		"SECURITY_PRIVILEGED",
		"SECURITY_CAPABILITIES",
		"SECURITY_CONFINEMENT",
		"SECURITY_DOCKER_SOCKET",
		"SECURITY_HOST_NETWORK",
		"SECURITY_HOST_PID",
		"SECURITY_HOST_IPC",
	}
	var got []string
	for _, f := range r.Findings {
		got = append(got, f.ID)
		if f.Level != "warning" || f.Service != "risky" {
			t.Fatalf("wrong finding classification: %+v", f)
		}
	}
	if !reflect.DeepEqual(got, want) || r.Summary.Warnings != len(want) || r.Summary.Blockers != 0 {
		t.Fatalf("findings = %v, summary = %+v; want %v", got, r.Summary, want)
	}
	reportJSON, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"SECRET_VALUE_NEVER_PRINT", "SECRET_BIND_PATH", "INTERNAL_TOKEN", "/run/user/1000"} {
		if strings.Contains(string(reportJSON), secret) {
			t.Fatalf("sensitive configuration leaked in report: %q", secret)
		}
	}
	if reportExitCode(r, errors.New("failed inspection"), false) != 2 {
		t.Fatal("inspection failure must preserve exit code 2")
	}
	if reportExitCode(r, nil, false) != 0 || reportExitCode(r, nil, true) != 1 {
		t.Fatal("security warnings must stay warnings; only --strict may fail them")
	}
}

func TestSecurityWarningsIgnoreCommonConfig(t *testing.T) {
	const resolved = `{"name":"normal","services":{"api":{
		"cap_add":["CHOWN","NET_BIND_SERVICE"],
		"security_opt":["no-new-privileges:true","seccomp=default.json"],
		"network_mode":"bridge","pid":"private","ipc":"private",
		"volumes":[{"type":"volume","source":"docker.sock","target":"/var/run/docker.sock"}]
	}}}`
	var m model
	if err := json.Unmarshal([]byte(resolved), &m); err != nil {
		t.Fatal(err)
	}
	r := Report{}
	securityWarnings(m, nil, "", &r)
	if len(r.Findings) != 0 {
		t.Fatalf("normal configuration must not trigger warnings: %+v", r.Findings)
	}
}

func TestSecurityWarningsRespectProfilesAndOrder(t *testing.T) {
	m := model{Services: map[string]service{
		"zeta":  {Privileged: true, Profiles: []string{"debug"}},
		"beta":  {Privileged: true},
		"alpha": {Privileged: true},
	}}
	tests := []struct {
		name     string
		selected []string
		fromEnv  string
		want     []string
	}{
		{"default", nil, "", []string{"alpha", "beta"}},
		{"explicit", []string{"debug"}, "", []string{"alpha", "beta", "zeta"}},
		{"environment", nil, "debug", []string{"alpha", "beta", "zeta"}},
		{"wildcard", []string{"*"}, "", []string{"alpha", "beta", "zeta"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var r Report
			securityWarnings(m, tc.selected, tc.fromEnv, &r)
			var got []string
			for _, f := range r.Findings {
				got = append(got, f.Service)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("services = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSecurityWarningsRemoteEngineWithoutHostProbes(t *testing.T) {
	h := Host{
		Run: func(_ context.Context, name string, args ...string) ([]byte, error) {
			switch {
			case name == "docker" && len(args) > 0 && args[0] == "compose":
				return []byte(`{"name":"remote","services":{"app":{"image":"alpine","privileged":true}}}`), nil
			case name == "docker" && len(args) > 0 && args[0] == "context":
				return []byte("tcp://remote.example.invalid:2376"), nil
			default:
				t.Fatalf("unexpected host command: %s %v", name, args)
				return nil, nil
			}
		},
		Getenv: func(string) string { return "" },
	}
	r, err := inspectForOS(context.Background(), Options{}, h, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if !hasID(r, "SECURITY_PRIVILEGED") || !hasID(r, "CONTEXT") {
		t.Fatalf("resolved Compose checks should work for remote contexts: %+v", r.Findings)
	}
	if hasID(r, "DAEMON") {
		t.Fatal("host checks should remain skipped for remote contexts")
	}
}

func TestSecurityWarningPredicateBoundaries(t *testing.T) {
	for _, c := range []string{"ALL", "cap_sys_admin", "CAP_NET_ADMIN", "SYS_PTRACE", "SYS_RAWIO", "BPF"} {
		if !hasElevatedCapability([]string{c}) {
			t.Errorf("high-impact capability %q was ignored", c)
		}
	}
	for _, c := range []string{"NET_BIND_SERVICE", "CHOWN", "SETUID", "AUDIT_WRITE"} {
		if hasElevatedCapability([]string{c}) {
			t.Errorf("ordinary capability %q was flagged", c)
		}
	}
	for _, option := range []string{"seccomp=unconfined", "apparmor=unconfined", "label=disable", "seccomp:unconfined"} {
		if !disablesConfinement([]string{option}) {
			t.Errorf("disabled confinement %q was ignored", option)
		}
	}
	for _, option := range []string{"no-new-privileges:true", "seccomp=profile.json", "apparmor=my-profile"} {
		if disablesConfinement([]string{option}) {
			t.Errorf("configured confinement %q was flagged", option)
		}
	}
}

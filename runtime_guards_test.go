package main

import (
	"context"
	"strings"
	"testing"
)

func runtimeGuardHost(config, version string) Host {
	h := fakeHost(config)
	previous := h.Run
	h.Run = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		if command == "docker" && strings.Join(args, " ") == "compose version --short" {
			return []byte(version), nil
		}
		return previous(ctx, command, args...)
	}
	return h
}
func TestRuntimeGuardsAffectedCompose(t *testing.T) {
	cases := []struct{ name, value, want string }{
		{"prestart-replica", "\"pre_start\":[{\"per_replica\":true}],\"deploy\":{\"replicas\":3}", "PRE_START_REPLICA_UNSUPPORTED"},
		{"restart-delay", "\"restart\":\"always\",\"deploy\":{\"restart_policy\":{\"delay\":\"5s\"}}", "RESTART_POLICY_OVERRIDDEN"},
		{"restart-max", "\"restart\":\"always\",\"deploy\":{\"restart_policy\":{\"max_attempts\":3}}", "RESTART_POLICY_INVALID"},
		{"restart-onfailure-delay", "\"restart\":\"on-failure\",\"deploy\":{\"restart_policy\":{\"delay\":\"5s\"}}", "RESTART_POLICY_OVERRIDDEN"},
		{"restart-unless-delay", "\"restart\":\"unless-stopped\",\"deploy\":{\"restart_policy\":{\"delay\":\"5s\"}}", "RESTART_POLICY_OVERRIDDEN"},
		{"max-no-restart", "\"deploy\":{\"restart_policy\":{\"max_attempts\":3}}", "RESTART_POLICY_INVALID"},
		{"any-max-invalid", "\"deploy\":{\"restart_policy\":{\"condition\":\"any\",\"max_attempts\":3}}", "RESTART_POLICY_INVALID"},
		{"empty-override", "\"restart\":\"always\",\"deploy\":{\"restart_policy\":{}}", "RESTART_POLICY_OVERRIDDEN"},
	}
	for _, c := range cases {
		for _, version := range []string{"v5.5.0", "v5.5.1", "5.6.0", "v5.6.1\n"} {
			t.Run(c.name+"/"+version, func(t *testing.T) {
				raw := "{\"name\":\"case\",\"services\":{\"app\":{" + c.value + "}}}"
				r, e := inspect(context.Background(), Options{}, runtimeGuardHost(raw, version))
				if e != nil {
					t.Fatal(e)
				}
				f, ok := find(r, c.want)
				if !ok || f.Level != "warning" || f.Service != "app" {
					t.Fatalf("missing %s: %+v", c.want, r)
				}
				if r.Summary.Blockers != 0 || reportExitCode(r, nil, false) != 0 || reportExitCode(r, nil, true) != 1 {
					t.Fatalf("status %+v", r)
				}
			})
		}
	}
}
func TestRuntimeGuardsNoFalseWarnings(t *testing.T) {
	cases := []struct{ name, value, version string }{
		{"normal-hook", "\"pre_start\":[{\"per_replica\":false}]", "5.5.1"},
		{"normal-restart", "\"restart\":\"always\"", "5.5.1"},
		{"explicit-any", "\"restart\":\"always\",\"deploy\":{\"restart_policy\":{\"condition\":\"any\",\"delay\":\"5s\"}}", "5.5.1"},
		{"explicit-onfailure", "\"restart\":\"on-failure\",\"deploy\":{\"restart_policy\":{\"condition\":\"on-failure\",\"max_attempts\":3}}", "5.5.1"},
		{"newer-unknown", "\"restart\":\"always\",\"pre_start\":[{\"per_replica\":true}],\"deploy\":{\"restart_policy\":{\"delay\":\"5s\"}}", "5.7.0"},
		{"older-unknown", "\"restart\":\"always\",\"pre_start\":[{\"per_replica\":true}],\"deploy\":{\"restart_policy\":{\"delay\":\"5s\"}}", "2.39.0"},
		{"invalid-version", "\"restart\":\"always\",\"pre_start\":[{\"per_replica\":true}],\"deploy\":{\"restart_policy\":{\"delay\":\"5s\"}}", "unknown"},
		{"inactive-profile", "\"restart\":\"always\",\"pre_start\":[{\"per_replica\":true}],\"deploy\":{\"restart_policy\":{\"delay\":\"5s\"}},\"profiles\":[\"debug\"]", "5.5.1"},
		{"only-delay", "\"deploy\":{\"restart_policy\":{\"delay\":\"5s\"}}", "5.5.1"},
		{"no-restart-delay", "\"restart\":\"no\",\"deploy\":{\"restart_policy\":{\"delay\":\"5s\"}}", "5.5.1"},
		{"max-zero", "\"restart\":\"always\",\"deploy\":{\"restart_policy\":{\"max_attempts\":0,\"condition\":\"any\"}}", "5.5.1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			raw := "{\"name\":\"case\",\"services\":{\"app\":{" + c.value + "}}}"
			r, e := inspect(context.Background(), Options{}, runtimeGuardHost(raw, c.version))
			if e != nil {
				t.Fatal(e)
			}
			if hasID(r, "PRE_START_REPLICA_UNSUPPORTED") || hasID(r, "RESTART_POLICY_OVERRIDDEN") || hasID(r, "RESTART_POLICY_INVALID") {
				t.Fatalf("unexpected warning %+v", r.Findings)
			}
		})
	}
}
func TestRuntimeGuardsProfileAndSorting(t *testing.T) {
	raw := "{\"name\":\"case\",\"services\":{\"z\":{\"profiles\":[\"test\"],\"pre_start\":[{\"per_replica\":true}]},\"a\":{\"restart\":\"always\",\"deploy\":{\"restart_policy\":{\"delay\":\"5s\"}}},\"b\":{\"pre_start\":[{\"per_replica\":true}]}}}"
	r, e := inspect(context.Background(), Options{Profiles: []string{"test"}}, runtimeGuardHost(raw, "5.5.1"))
	if e != nil {
		t.Fatal(e)
	}
	var found []string
	for _, f := range r.Findings {
		if f.ID == "PRE_START_REPLICA_UNSUPPORTED" || f.ID == "RESTART_POLICY_OVERRIDDEN" {
			found = append(found, f.Service+":"+f.ID)
		}
	}
	want := []string{"a:RESTART_POLICY_OVERRIDDEN", "b:PRE_START_REPLICA_UNSUPPORTED", "z:PRE_START_REPLICA_UNSUPPORTED"}
	if strings.Join(found, ";") != strings.Join(want, ";") {
		t.Fatalf("got %v want %v", found, want)
	}
}

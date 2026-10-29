package main

import (
	"context"
	"strings"
	"testing"
)

func replicaHost(raw string) Host {
	h := fakeHost(raw)
	old := h.Run
	h.Run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		q := name + " " + strings.Join(args, " ")
		if strings.Contains(q, "docker ps --filter label=com.docker.compose.project=scale-demo") {
			return []byte("abcabcabcabc|worker\ndefdefdefdef|worker\na1a1a1a1a1a1|api\n"), nil
		}
		if strings.Contains(q, "docker inspect --format") {
			return []byte(`{}|||[]`), nil
		}
		return old(ctx, name, args...)
	}
	return h
}

func TestScaledServiceDoesNotHideUnrelatedComparison(t *testing.T) {
	raw := `{"name":"scale-demo","services":{"api":{"ports":[]},"worker":{"ports":[]}}}`
	r, e := inspect(context.Background(), Options{Compare: true}, replicaHost(raw))
	if e != nil {
		t.Fatal(e)
	}
	if !hasID(r, "REPLICAS_UNSUPPORTED") {
		t.Fatalf("expected explicit replica warning: %+v", r)
	}
	if hasID(r, "BASELINE_UNKNOWN") {
		t.Fatalf("one scaled service must not hide all checks: %+v", r)
	}
	if hasID(r, "SERVICE_NEW") {
		t.Fatalf("scaled worker is not a new service: %+v", r)
	}
	if hasID(r, "RUNNING_MATCH") {
		t.Fatalf("partial result must not claim complete match: %+v", r)
	}
}
func TestOnlyScaledServiceDoesNotClaimMissingStack(t *testing.T) {
	raw := `{"name":"scale-demo","services":{"worker":{"ports":[]}}}`
	h := replicaHost(raw)
	old := h.Run
	h.Run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		q := name + " " + strings.Join(args, " ")
		if strings.Contains(q, "docker ps --filter label=com.docker.compose.project=scale-demo") {
			return []byte("abcabcabcabc|worker\ndefdefdefdef|worker\n"), nil
		}
		return old(ctx, name, args...)
	}
	r, e := inspect(context.Background(), Options{Compare: true}, h)
	if e != nil {
		t.Fatal(e)
	}
	if !hasID(r, "REPLICAS_UNSUPPORTED") || hasID(r, "NO_RUNNING_STACK") {
		t.Fatalf("replicas are running: %+v", r)
	}
}

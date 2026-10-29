package main

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestDockerPortOwnerLookupFailureIsNotReportedAsClear(t *testing.T) {
	h := fakeHost(`{"name":"demo","services":{"web":{"ports":[{"published":"27051","protocol":"tcp"}]}}}`)
	old := h.Run
	h.Run = func(ctx context.Context, cmd string, args ...string) ([]byte, error) {
		joined := cmd + " " + strings.Join(args, " ")
		if strings.Contains(joined, "docker ps --filter publish=") {
			return nil, errors.New("docker publish API unavailable")
		}
		if strings.HasPrefix(joined, "ss -H") {
			return []byte(""), nil
		}
		return old(ctx, cmd, args...)
	}
	report, err := inspect(context.Background(), Options{}, h)
	if err != nil {
		t.Fatal(err)
	}
	if !hasID(report, "PORT_OWNERS_UNKNOWN") {
		t.Fatalf("unavailable Docker port lookup silently passed: %+v", report)
	}
	if hasID(report, "PORTS") {
		t.Fatalf("port availability was claimed after owner lookup failed: %+v", report)
	}
}

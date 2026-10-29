package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestBuildContextMissingIsWarnedWithoutLeakingPath(t *testing.T) {
	secretPath := "/tmp/private-build-context-token"
	raw := `{"name":"build-app","services":{"api":{"build":{"context":"` + secretPath + `"}}}}`
	r, e := inspect(context.Background(), Options{}, fakeHost(raw))
	if e != nil {
		t.Fatal(e)
	}
	finding, ok := find(r, "BUILD_CONTEXT_MISSING")
	if !ok || finding.Level != "warning" {
		t.Fatalf("missing local build context not flagged: %+v", r)
	}
	if strings.Contains(fmt.Sprint(r), secretPath) {
		t.Fatalf("local build path leaked in finding")
	}
	if r.Summary.Blockers != 0 {
		t.Fatalf("warning should not be a blocker: %+v", r)
	}
}
func TestExistingBuildContextDoesNotWarn(t *testing.T) {
	h := fakeHost(`{"name":"build-app","services":{"api":{"build":{"context":"/tmp/available"}}}}`)
	h.Stat = func(string) (os.FileInfo, error) { return nil, nil }
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil {
		t.Fatal(e)
	}
	if hasID(r, "BUILD_CONTEXT_MISSING") || hasID(r, "BUILD_CONTEXT_UNKNOWN") {
		t.Fatalf("unexpected build context warning %+v", r)
	}
}
func TestRemoteBuildContextDoesNotProbeLocalPath(t *testing.T) {
	h := fakeHost(`{"name":"build-app","services":{"api":{"build":{"context":"https://github.com/example/private.git"}}}}`)
	old := h.Stat
	h.Stat = func(path string) (os.FileInfo, error) {
		if strings.Contains(path, "github.com") {
			t.Fatalf("remote build context inspected as local path")
		}
		return old(path)
	}
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil {
		t.Fatal(e)
	}
	if hasID(r, "BUILD_CONTEXT_MISSING") {
		t.Fatalf("remote build context falsely missing %+v", r)
	}
}

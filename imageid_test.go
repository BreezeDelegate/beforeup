package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func hostForLocalImage(plan, runningID, localID string) Host {
	h := fakeHost(plan)
	old := h.Run
	h.Run = func(ctx context.Context, cmd string, args ...string) ([]byte, error) {
		joined := cmd + " " + strings.Join(args, " ")
		if strings.Contains(joined, "docker ps --filter label=com.docker.compose.project=demo") {
			return []byte("deadbeef1234|web\n"), nil
		}
		if strings.Contains(joined, "docker inspect --format") && strings.Contains(joined, "deadbeef1234") {
			if strings.Contains(joined, ".Config.Image") {
				return []byte(`"example/web:stable"|||[]`), nil
			}
			if strings.Contains(joined, ".Image") {
				return []byte(runningID + "\n"), nil
			}
			return []byte(`{}|||[]`), nil
		}
		if strings.Contains(joined, "docker image inspect --format") {
			if localID == "error" {
				return nil, fmt.Errorf("no local image")
			}
			return []byte(localID + "\n"), nil
		}
		return old(ctx, cmd, args...)
	}
	return h
}
func TestSameTagDifferentLocalImageIsDetected(t *testing.T) {
	plan := `{"name":"demo","services":{"web":{"image":"example/web:stable"}}}`
	h := hostForLocalImage(plan, "sha256:"+strings.Repeat("1", 64), "sha256:"+strings.Repeat("2", 64))
	r, e := inspect(context.Background(), Options{Compare: true}, h)
	if e != nil {
		t.Fatal(e)
	}
	if !hasID(r, "LOCAL_IMAGE_UPDATED") || hasID(r, "RUNNING_MATCH") {
		t.Fatalf("local tag moved but not detected: %+v", r)
	}
	if strings.Contains(fmt.Sprint(r), "sha256:111") || strings.Contains(fmt.Sprint(r), "sha256:222") {
		t.Fatal("image IDs leaked")
	}
}
func TestSameLocalImageDoesNotWarn(t *testing.T) {
	plan := `{"name":"demo","services":{"web":{"image":"example/web:stable"}}}`
	h := hostForLocalImage(plan, "sha256:"+strings.Repeat("1", 64), "sha256:"+strings.Repeat("1", 64))
	r, e := inspect(context.Background(), Options{Compare: true}, h)
	if e != nil {
		t.Fatal(e)
	}
	if hasID(r, "LOCAL_IMAGE_UPDATED") || !hasID(r, "RUNNING_MATCH") {
		t.Fatalf("false local update: %+v", r)
	}
}
func TestUnavailableLocalImageIsNotClaimedUpdated(t *testing.T) {
	plan := `{"name":"demo","services":{"web":{"image":"example/web:stable"}}}`
	h := hostForLocalImage(plan, "sha256:"+strings.Repeat("1", 64), "error")
	r, e := inspect(context.Background(), Options{Compare: true}, h)
	if e != nil {
		t.Fatal(e)
	}
	if hasID(r, "LOCAL_IMAGE_UPDATED") {
		t.Fatalf("missing image is not a proven update: %+v", r)
	}
}

func TestPinnedPlatformSkipsLocalImageIdentityCheck(t *testing.T) {
	plan := `{"name":"demo","services":{"web":{"image":"example/web:stable","platform":"linux/arm64"}}}`
	h := hostForLocalImage(plan, "sha256:"+strings.Repeat("1", 64), "sha256:"+strings.Repeat("2", 64))
	old := h.Run
	h.Run = func(ctx context.Context, cmd string, args ...string) ([]byte, error) {
		if cmd == "docker" && len(args) >= 2 && args[0] == "image" && args[1] == "inspect" {
			t.Fatal("platform-sensitive local image inspect must be skipped")
		}
		return old(ctx, cmd, args...)
	}
	r, e := inspect(context.Background(), Options{Compare: true}, h)
	if e != nil {
		t.Fatal(e)
	}
	if hasID(r, "LOCAL_IMAGE_UPDATED") {
		t.Fatalf("pinned platform should not compare host-default image IDs: %+v", r)
	}
}

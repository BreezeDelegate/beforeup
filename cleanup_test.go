package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestCompareReportsScaledOrphanWithoutLeakingData(t *testing.T) {
	raw := `{"name":"demo","services":{"web":{"image":"app:test"}}}`
	h := fakeHost(raw)
	old := h.Run
	h.Run = func(ctx context.Context, cmd string, args ...string) ([]byte, error) {
		c := cmd + " " + strings.Join(args, " ")
		if strings.Contains(c, "docker ps --filter label=com.docker.compose.project=demo") {
			return []byte("abcabcabcabc|old\ndefdefdefdef|old\na1a1a1a1a1a1|web\n"), nil
		}
		if strings.Contains(c, "docker inspect --format") {
			if strings.Contains(c, ".Config.Image") {
				return []byte(`"app:test"|||[]`), nil
			}
			return []byte(`{}|||[]`), nil
		}
		return old(ctx, cmd, args...)
	}
	report, err := inspect(context.Background(), Options{Compare: true}, h)
	if err != nil {
		t.Fatal(err)
	}
	if !hasID(report, "SERVICE_ABSENT_FROM_PROPOSAL") {
		t.Fatalf("scaled orphan was silently omitted: %+v", report)
	}
	if hasID(report, "RUNNING_MATCH") {
		t.Fatalf("scaled orphan cannot claim clean comparison: %+v", report)
	}
}

func TestInspectDoesNotExposeInterpolatedMountTarget(t *testing.T) {
	raw := `{"name":"demo","services":{"web":{"image":"app:test"}}}`
	h := fakeHost(raw)
	old := h.Run
	h.Run = func(ctx context.Context, cmd string, args ...string) ([]byte, error) {
		c := cmd + " " + strings.Join(args, " ")
		if strings.Contains(c, "docker ps --filter label=com.docker.compose.project=demo") {
			return []byte("abcabcabcabc|web\n"), nil
		}
		if strings.Contains(c, "docker inspect --format") {
			if strings.Contains(c, ".Config.Image") {
				return []byte(`"app:test"|||[]`), nil
			}
			return []byte(`{}|||[{"Type":"bind","Source":"/host","Destination":"/var/private-value-xyz"}]`), nil
		}
		return old(ctx, cmd, args...)
	}
	report, err := inspect(context.Background(), Options{Compare: true}, h)
	if err != nil {
		t.Fatal(err)
	}
	if !hasID(report, "MOUNT_REMOVED") {
		t.Fatalf("missing mount change warning %+v", report)
	}
	if strings.Contains(fmt.Sprint(report), "private-value-xyz") {
		t.Fatal("interpolated mount target leaked into report")
	}
}

func TestCompareScaledReplicasNeedNoInspect(t *testing.T) {
	raw := `{"name":"demo","services":{"worker":{"image":"app:test"}}}`
	h := fakeHost(raw)
	old := h.Run
	inspectCount := 0
	h.Run = func(ctx context.Context, cmd string, args ...string) ([]byte, error) {
		c := cmd + " " + strings.Join(args, " ")
		if strings.Contains(c, "docker ps --filter label=com.docker.compose.project=demo") {
			return []byte("abcabcabcabc|worker\ndefdefdefdef|worker\n"), nil
		}
		if strings.Contains(c, "docker inspect --format") {
			inspectCount++
			return nil, fmt.Errorf("inspect denied")
		}
		return old(ctx, cmd, args...)
	}
	report, err := inspect(context.Background(), Options{Compare: true}, h)
	if err != nil {
		t.Fatal(err)
	}
	if !hasID(report, "REPLICAS_UNSUPPORTED") || hasID(report, "BASELINE_UNKNOWN") {
		t.Fatalf("scaled service should not trigger an unnecessary inspect: %+v", report)
	}
	if inspectCount != 0 {
		t.Fatalf("unnecessary inspect commands: %d", inspectCount)
	}
}

func TestCompareOrphanFindingsAreDeterministicallyOrdered(t *testing.T) {
	raw := `{"name":"demo","services":{"web":{}}}`
	h := fakeHost(raw)
	old := h.Run
	h.Run = func(ctx context.Context, cmd string, args ...string) ([]byte, error) {
		c := cmd + " " + strings.Join(args, " ")
		if strings.Contains(c, "docker ps --filter label=com.docker.compose.project=demo") {
			return []byte("aaaabbbbcccc|oldz\nddddeeeeffff|oldb\n123456789abc|web\n"), nil
		}
		if strings.Contains(c, "docker inspect --format") {
			return []byte(`{}|||[]`), nil
		}
		return old(ctx, cmd, args...)
	}
	for i := 0; i < 50; i++ {
		report, err := inspect(context.Background(), Options{Compare: true}, h)
		if err != nil {
			t.Fatal(err)
		}
		orphan := []string{}
		for _, f := range report.Findings {
			if f.ID == "SERVICE_ABSENT_FROM_PROPOSAL" {
				orphan = append(orphan, f.Service)
			}
		}
		if len(orphan) != 2 || orphan[0] != "oldb" || orphan[1] != "oldz" {
			t.Fatalf("unordered orphan reports: %v", orphan)
		}
	}
}

func TestCompareSummarizesMountChangesByService(t *testing.T) {
	raw := `{"name":"demo","services":{"web":{"volumes":[{"type":"bind","source":"/new-a","target":"/app/a-private-one"},{"type":"bind","source":"/new-b","target":"/app/b-private-two"}]}}}`
	h := fakeHost(raw)
	old := h.Run
	h.Run = func(ctx context.Context, cmd string, args ...string) ([]byte, error) {
		c := cmd + " " + strings.Join(args, " ")
		if strings.Contains(c, "docker ps --filter label=com.docker.compose.project=demo") {
			return []byte("abcabcabcabc|web\n"), nil
		}
		if strings.Contains(c, "docker inspect --format") {
			return []byte(`{}|||[{"Type":"bind","Source":"/old-a","Destination":"/app/a-private-one"},{"Type":"bind","Source":"/old-b","Destination":"/app/b-private-two"},{"Type":"bind","Source":"/old-c","Destination":"/app/c-private-three"}]`), nil
		}
		return old(ctx, cmd, args...)
	}
	r, e := inspect(context.Background(), Options{Compare: true}, h)
	if e != nil {
		t.Fatal(e)
	}
	if countFindings(r, "MOUNT_SOURCE_CHANGED") != 1 || countFindings(r, "MOUNT_REMOVED") != 1 {
		t.Fatalf("expected concise mount summary, got %+v", r)
	}
	changed, _ := find(r, "MOUNT_SOURCE_CHANGED")
	if !strings.Contains(changed.Message, "2") {
		t.Fatalf("two changed mounts not counted: %+v", changed)
	}
	for _, private := range []string{"private-one", "private-two", "private-three", "/old-a", "/new-a"} {
		if strings.Contains(fmt.Sprint(r), private) {
			t.Fatalf("mount source or target leaked: %s", private)
		}
	}
}

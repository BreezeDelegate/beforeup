package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
)

const changedModel = `{"name":"demo","services":{"web":{"ports":[{"published":"8080","host_ip":"127.0.0.1","protocol":"tcp"}],"volumes":[{"type":"bind","source":"/new/data","target":"/data","bind":{"create_host_path":false}}]}}}`

func comparisonHost(plan string) Host {
	h := fakeHost(plan)
	h.Stat = func(path string) (os.FileInfo, error) { return nil, nil }
	old := h.Run
	h.Run = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		v := command + " " + strings.Join(args, " ")
		switch {
		case strings.Contains(v, "docker ps --filter label=com.docker.compose.project=demo"):
			return []byte("deadbeef1234|web\n"), nil
		case strings.Contains(v, "docker inspect --format") && strings.Contains(v, "deadbeef1234"):
			return []byte(`{"80/tcp":[{"HostIp":"127.0.0.1","HostPort":"8081"}]}|||[{"Type":"bind","Source":"/old/data","Destination":"/data"}]`), nil
		case strings.Contains(v, "docker ps --filter publish="):
			return []byte(""), nil
		default:
			return old(ctx, command, args...)
		}
	}
	return h
}
func TestCompareReportsChangesToRunningService(t *testing.T) {
	r, e := inspect(context.Background(), Options{Compare: true}, comparisonHost(changedModel))
	if e != nil {
		t.Fatal(e)
	}
	for _, id := range []string{"PORT_ADDED", "PORT_REMOVED", "MOUNT_SOURCE_CHANGED"} {
		if !hasID(r, id) {
			t.Errorf("missing %s: %+v", id, r)
		}
	}
}
func TestCompareHasNoFabricatedBaseline(t *testing.T) {
	h := comparisonHost(changedModel)
	old := h.Run
	h.Run = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		v := command + " " + strings.Join(args, " ")
		if strings.Contains(v, "docker ps --filter label=com.docker.compose.project=demo") {
			return []byte(""), nil
		}
		return old(ctx, command, args...)
	}
	r, e := inspect(context.Background(), Options{Compare: true}, h)
	if e != nil {
		t.Fatal(e)
	}
	if !hasID(r, "NO_RUNNING_STACK") || hasID(r, "MOUNT_SOURCE_CHANGED") {
		t.Fatalf("invented diff: %+v", r)
	}
}
func TestExistingPortOwnedBySameServiceIsNotAConflict(t *testing.T) {
	h := fakeHost(`{"name":"demo","services":{"web":{"ports":[{"published":"8080","host_ip":"127.0.0.1","protocol":"tcp"}]}}}`)
	old := h.Run
	h.Run = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		v := command + " " + strings.Join(args, " ")
		if strings.Contains(v, "docker ps --filter publish=8080") {
			return []byte("abcdef123456|demo|web\n"), nil
		}
		if strings.Contains(v, "docker inspect --format") && strings.Contains(v, "abcdef123456") {
			return []byte(`{"80/tcp":[{"HostIp":"127.0.0.1","HostPort":"8080"}]}|||[]`), nil
		}
		return old(ctx, command, args...)
	}
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil {
		t.Fatal(e)
	}
	if hasID(r, "PORT_IN_USE") || hasID(r, "PORT_USED_BY_OTHER") {
		t.Fatalf("own port incorrectly treated as conflict: %+v", r)
	}
	if !hasID(r, "PORT_OWNED_BY_SERVICE") {
		t.Fatalf("ownership missing: %+v", r)
	}
}
func TestDockerPublishedPortVisibleWithoutSocketListener(t *testing.T) {
	h := fakeHost(`{"name":"demo","services":{"web":{"ports":[{"published":"8080","host_ip":"127.0.0.1","protocol":"tcp"}]}}}`)
	old := h.Run
	h.Run = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		v := command + " " + strings.Join(args, " ")
		if strings.Contains(v, "docker ps --filter publish=8080") {
			return []byte("abcdef123456|another-stack|api\n"), nil
		}
		if strings.Contains(v, "docker inspect --format") && strings.Contains(v, "abcdef123456") {
			return []byte(`{"80/tcp":[{"HostIp":"127.0.0.1","HostPort":"8080"}]}|||[]`), nil
		}
		if strings.HasPrefix(v, "ss -H") {
			return []byte(""), nil
		}
		return old(ctx, command, args...)
	}
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil {
		t.Fatal(e)
	}
	finding, ok := find(r, "PORT_USED_BY_OTHER")
	if !ok || !strings.Contains(finding.Message, "another-stack") {
		t.Fatalf("docker published conflict not detected: %+v", r)
	}
}
func TestComparisonFailureIsExplicit(t *testing.T) {
	h := comparisonHost(changedModel)
	old := h.Run
	h.Run = func(ctx context.Context, command string, args ...string) ([]byte, error) {
		v := command + " " + strings.Join(args, " ")
		if strings.Contains(v, "docker ps --filter label=com.docker.compose.project=demo") {
			return nil, fmt.Errorf("Docker denied")
		}
		return old(ctx, command, args...)
	}
	r, e := inspect(context.Background(), Options{Compare: true}, h)
	if e != nil {
		t.Fatal(e)
	}
	if !hasID(r, "BASELINE_UNKNOWN") {
		t.Fatalf("comparison failure silently ignored: %+v", r)
	}
}

func TestNamedVolumeUsesResolvedComposeName(t *testing.T) {
	planned := `{"name":"demo-stack","volumes":{"data_volume":{"name":"demo-stack_data_volume"}},"services":{"db":{"volumes":[{"type":"volume","source":"data_volume","target":"/var/lib/mysql"}]}}}`
	h := fakeHost(planned)
	old := h.Run
	h.Run = func(ctx context.Context, cmd string, args ...string) ([]byte, error) {
		joined := cmd + " " + strings.Join(args, " ")
		if strings.Contains(joined, "docker ps --filter label=com.docker.compose.project=demo-stack") {
			return []byte("deadbeef1234|db\n"), nil
		}
		if strings.Contains(joined, "docker inspect --format") && strings.Contains(joined, "deadbeef1234") {
			return []byte(`{}|||[{"Type":"volume","Name":"demo-stack_data_volume","Source":"/var/lib/docker/volumes/demo-stack_data_volume/_data","Destination":"/var/lib/mysql"}]`), nil
		}
		return old(ctx, cmd, args...)
	}
	r, e := inspect(context.Background(), Options{Compare: true}, h)
	if e != nil {
		t.Fatal(e)
	}
	if hasID(r, "MOUNT_SOURCE_CHANGED") {
		t.Fatalf("normal Docker volume naming mistaken for change: %+v", r)
	}
	if !hasID(r, "RUNNING_MATCH") {
		t.Fatalf("expected matching named volume: %+v", r)
	}
}

func TestResolvedBindSourceIsNotPrinted(t *testing.T) {
	model := `{"name":"sample","services":{"api":{"volumes":[{"type":"bind","source":"/tmp/session-PRIVATE_TOKEN_DO_NOT_PRINT","target":"/mnt/data","bind":{"create_host_path":false}}]}}}`
	h := fakeHost(model)
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil {
		t.Fatal(e)
	}
	if !hasID(r, "BIND_MISSING") {
		t.Fatalf("expected missing bind: %+v", r)
	}
	if strings.Contains(fmt.Sprint(r), "PRIVATE_TOKEN_DO_NOT_PRINT") {
		t.Fatalf("resolved path leaked credential-like value: %+v", r)
	}
}

func TestComposeWithNoActiveProfilesIsNotAnError(t *testing.T) {
	h := fakeHost(`{"name":"profile-only","services":{}}`)
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil || r.Summary.Blockers > 0 || !hasID(r, "NO_ACTIVE_SERVICES") {
		t.Fatalf("profile-only project should not fail validation: %+v / %v", r, e)
	}
}

func TestWildcardComposeProfileActivatesServices(t *testing.T) {
	if !profileEnabled([]string{"debug"}, []string{"*"}, "") {
		t.Fatal("wildcard must enable profile")
	}
	if !profileEnabled([]string{"debug"}, nil, "*") {
		t.Fatal("wildcard from COMPOSE_PROFILES must enable profile")
	}
}
func TestOwnPortOnDifferentHostAddressIsAChange(t *testing.T) {
	h := fakeHost(`{"name":"demo","services":{"api":{"ports":[{"published":"8080","host_ip":"127.0.0.1","protocol":"tcp"}]}}}`)
	old := h.Run
	h.Run = func(ctx context.Context, cmd string, args ...string) ([]byte, error) {
		joined := cmd + " " + strings.Join(args, " ")
		if strings.Contains(joined, "docker ps --filter publish=8080") {
			return []byte("abcdef123456|demo|api\n"), nil
		}
		if strings.Contains(joined, "docker inspect --format") && strings.Contains(joined, "abcdef123456") {
			return []byte(`{"80/tcp":[{"HostIp":"0.0.0.0","HostPort":"8080"}]}|||[]`), nil
		}
		return old(ctx, cmd, args...)
	}
	r, e := inspect(context.Background(), Options{}, h)
	if e != nil {
		t.Fatal(e)
	}
	if !hasID(r, "PORT_BINDING_CHANGED") || hasID(r, "PORT_USED_BY_OTHER") {
		t.Fatalf("own address change was misattributed: %+v", r)
	}
}

func TestCompareDetectsChangedEnvironmentWithoutValues(t *testing.T) {
	raw := `{"name":"demo","services":{"web":{"image":"example/web:1","environment":{"PASSWORD":"new-secret-abc","MODE":"production"}}}}`
	h := fakeHost(raw)
	old := h.Run
	h.Run = func(ctx context.Context, cmd string, args ...string) ([]byte, error) {
		q := cmd + " " + strings.Join(args, " ")
		if strings.Contains(q, "docker ps --filter label=com.docker.compose.project=demo") {
			return []byte("deadbeef1234|web\n"), nil
		}
		if strings.Contains(q, "docker inspect --format") && strings.Contains(q, "deadbeef1234") {
			if strings.Contains(q, ".Config.Image") {
				return []byte(`"example/web:1"|||["PASSWORD=old-secret-xyz","MODE=production","BASE_IMAGE_VAR=safe"]`), nil
			}
			return []byte(`{}|||[]`), nil
		}
		return old(ctx, cmd, args...)
	}
	r, e := inspect(context.Background(), Options{Compare: true}, h)
	if e != nil {
		t.Fatal(e)
	}
	f, ok := find(r, "ENV_CHANGED")
	if !ok || f.Level != "warning" {
		t.Fatalf("expected environment warning %+v", r)
	}
	report := fmt.Sprint(r)
	for _, secret := range []string{"new-secret-abc", "old-secret-xyz", "PASSWORD"} {
		if strings.Contains(report, secret) {
			t.Fatalf("secret/name exposed %q", secret)
		}
	}
	if hasID(r, "RUNNING_MATCH") {
		t.Fatal("configuration change claimed to be identical")
	}
}
func TestCompareUnchangedEnvironmentDoesNotWarnAboutImageDefaults(t *testing.T) {
	raw := `{"name":"demo","services":{"web":{"image":"example/web:1","environment":{"MODE":"production"}}}}`
	h := fakeHost(raw)
	old := h.Run
	h.Run = func(ctx context.Context, cmd string, args ...string) ([]byte, error) {
		q := cmd + " " + strings.Join(args, " ")
		if strings.Contains(q, "docker ps --filter label=com.docker.compose.project=demo") {
			return []byte("deadbeef1234|web\n"), nil
		}
		if strings.Contains(q, "docker inspect --format") && strings.Contains(q, "deadbeef1234") {
			if strings.Contains(q, ".Config.Image") {
				return []byte(`"example/web:1"|||["MODE=production","PATH=/usr/bin","INHERITED=default"]`), nil
			}
			return []byte(`{}|||[]`), nil
		}
		return old(ctx, cmd, args...)
	}
	r, e := inspect(context.Background(), Options{Compare: true}, h)
	if e != nil {
		t.Fatal(e)
	}
	if hasID(r, "ENV_CHANGED") || hasID(r, "IMAGE_REFERENCE_CHANGED") || !hasID(r, "RUNNING_MATCH") {
		t.Fatalf("image defaults must not lead to false positives %+v", r)
	}
}
func TestCompareImageReferenceChangeDoesNotEchoReference(t *testing.T) {
	raw := `{"name":"demo","services":{"web":{"image":"example/web:private-value-987"}}}`
	h := fakeHost(raw)
	old := h.Run
	h.Run = func(ctx context.Context, cmd string, args ...string) ([]byte, error) {
		q := cmd + " " + strings.Join(args, " ")
		if strings.Contains(q, "docker ps --filter label=com.docker.compose.project=demo") {
			return []byte("deadbeef1234|web\n"), nil
		}
		if strings.Contains(q, "docker inspect --format") && strings.Contains(q, "deadbeef1234") {
			if strings.Contains(q, ".Config.Image") {
				return []byte(`"example/web:old"|||[]`), nil
			}
			return []byte(`{}|||[]`), nil
		}
		return old(ctx, cmd, args...)
	}
	r, e := inspect(context.Background(), Options{Compare: true}, h)
	if e != nil {
		t.Fatal(e)
	}
	if !hasID(r, "IMAGE_REFERENCE_CHANGED") || strings.Contains(fmt.Sprint(r), "private-value-987") {
		t.Fatalf("image change missing or leaked %+v", r)
	}
}
func TestCompareMissingRuntimeMetadataIsUnknown(t *testing.T) {
	raw := `{"name":"demo","services":{"web":{"image":"example/web:1","environment":{"MODE":"production"}}}}`
	h := fakeHost(raw)
	old := h.Run
	h.Run = func(ctx context.Context, cmd string, args ...string) ([]byte, error) {
		q := cmd + " " + strings.Join(args, " ")
		if strings.Contains(q, "docker ps --filter label=com.docker.compose.project=demo") {
			return []byte("deadbeef1234|web\n"), nil
		}
		if strings.Contains(q, "docker inspect --format") && strings.Contains(q, "deadbeef1234") {
			if strings.Contains(q, ".Config.Image") {
				return nil, fmt.Errorf("secret-value-in-error")
			}
			return []byte(`{}|||[]`), nil
		}
		return old(ctx, cmd, args...)
	}
	r, e := inspect(context.Background(), Options{Compare: true}, h)
	if e != nil {
		t.Fatal(e)
	}
	if !hasID(r, "RUNTIME_METADATA_UNKNOWN") || hasID(r, "RUNNING_MATCH") || strings.Contains(fmt.Sprint(r), "secret-value-in-error") {
		t.Fatalf("unknown baseline handling failed %+v", r)
	}
}
func TestCompareReportsAbsentServiceWithoutPredictingRemoval(t *testing.T) {
	raw := `{"name":"demo","services":{"new":{"image":"example/web:1"}}}`
	h := fakeHost(raw)
	old := h.Run
	h.Run = func(ctx context.Context, cmd string, args ...string) ([]byte, error) {
		q := cmd + " " + strings.Join(args, " ")
		if strings.Contains(q, "docker ps --filter label=com.docker.compose.project=demo") {
			return []byte("deadbeef1234|old\n"), nil
		}
		if strings.Contains(q, "docker inspect --format") && strings.Contains(q, "deadbeef1234") {
			return []byte(`{}|||[]`), nil
		}
		return old(ctx, cmd, args...)
	}
	r, e := inspect(context.Background(), Options{Compare: true}, h)
	if e != nil {
		t.Fatal(e)
	}
	f, ok := find(r, "SERVICE_ABSENT_FROM_PROPOSAL")
	if !ok || !strings.Contains(f.Advice, "orphan") {
		t.Fatalf("removed service must be reported as orphan risk, not certain deletion %+v", r)
	}
}

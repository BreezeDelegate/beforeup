package main

import (
	"context"
	"strings"
	"testing"
)

func TestComparisonPortFindingsAreDeterministic(t *testing.T) {
	plan := `{"name":"demo","services":{"web":{"ports":[{"published":"8082","host_ip":"127.0.0.1","protocol":"tcp"},{"published":"8080","host_ip":"127.0.0.1","protocol":"tcp"}]}}}`
	h := comparisonHost(plan)
	prev := h.Run
	h.Run = func(ctx context.Context, cmd string, args ...string) ([]byte, error) {
		joined := cmd + " " + strings.Join(args, " ")
		if strings.Contains(joined, "docker inspect --format") && strings.Contains(joined, "deadbeef1234") {
			return []byte(`{"80/tcp":[{"HostIp":"127.0.0.1","HostPort":"8083"},{"HostIp":"127.0.0.1","HostPort":"8081"}]}|||[]`), nil
		}
		return prev(ctx, cmd, args...)
	}
	for i := 0; i < 75; i++ {
		report, err := inspect(context.Background(), Options{Compare: true}, h)
		if err != nil {
			t.Fatal(err)
		}
		found := []string{}
		for _, f := range report.Findings {
			if f.ID == "PORT_ADDED" || f.ID == "PORT_REMOVED" {
				found = append(found, f.ID+":"+f.Message)
			}
		}
		if len(found) != 4 || !strings.Contains(found[0], "8080") || !strings.Contains(found[1], "8082") || !strings.Contains(found[2], "8081") || !strings.Contains(found[3], "8083") {
			t.Fatalf("port report order is not stable: %v", found)
		}
	}
}

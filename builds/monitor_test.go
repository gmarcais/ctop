package builds

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMonitorDiscoveryAndLifecycle(t *testing.T) {
	var mu sync.Mutex
	calls := make(map[string]int)
	finished := false
	m := &Monitor{run: func(ctx context.Context, args ...string) ([]byte, error) {
		mu.Lock()
		defer mu.Unlock()
		if args[1] == "ls" {
			// Duplicate entries must not result in duplicate queries/rows.
			return []byte(`{"Name":"default","Nodes":[{"Status":"running"}]}
{"Name":"named","Nodes":[{"Status":"running"}]}
{"Name":"named","Nodes":[{"Status":"running"}]}
{"Name":"stopped","Nodes":[{"Status":"inactive"}]}`), nil
		}
		name := args[2]
		calls[name]++
		if strings.Join(args[3:], " ") != "history ls --filter status=running --format json" {
			t.Errorf("unexpected query: %v", args)
		}
		if finished {
			return nil, nil
		}
		if name == "default" {
			return []byte(`{"ref":"default/default/a","name":"app","status":"Running","created_at":"2026-10-28T12:00:00Z","total_steps":8,"completed_steps":3,"cached_steps":2}
{"ref":"default/default/old","status":"Completed","completed_at":"2026-10-28T12:01:00Z"}
{"ref":"default/default/race","status":"Running","completed_at":"2026-10-28T12:01:00Z"}`), nil
		}
		return []byte(`[{"ref":"named/named0/b","name":"worker","status":"running","created_at":"2026-10-28T11:00:00Z"}]`), nil
	}}
	m.refresh(context.Background())
	rows, warning := m.Snapshot()
	if warning != "" || len(rows) != 2 {
		t.Fatalf("snapshot = %v, %q", rows, warning)
	}
	if rows[0].Builder != "named" || rows[1].CompletedSteps != 3 || rows[1].CachedSteps != 2 {
		t.Fatalf("incorrect sorting or metadata: %+v", rows)
	}
	if calls["named"] != 1 || calls["default"] != 1 || calls["stopped"] != 0 {
		t.Fatalf("unexpected builder queries: %v", calls)
	}
	rows[0].Name = "mutated"
	copyRows, _ := m.Snapshot()
	if copyRows[0].Name == "mutated" {
		t.Fatal("snapshot exposes shared mutable storage")
	}
	finished = true
	m.refresh(context.Background())
	rows, warning = m.Snapshot()
	if len(rows) != 0 || warning != "" {
		t.Fatalf("finished builds remain: %v, %q", rows, warning)
	}
}

func TestMonitorPartialFailureAndRecovery(t *testing.T) {
	fail := false
	m := &Monitor{run: func(ctx context.Context, args ...string) ([]byte, error) {
		if args[1] == "ls" {
			if fail {
				return nil, errors.New("docker unavailable")
			}
			return []byte(`[{"Name":"good","Nodes":[{"Status":"running"}]},{"Name":"old","Nodes":[{"Status":"running"}]}]`), nil
		}
		if args[2] == "old" {
			return nil, errors.New("build history unsupported")
		}
		return []byte(`{"ref":"good/node/ref","status":"Running"}`), nil
	}}
	m.refresh(context.Background())
	rows, warning := m.Snapshot()
	if len(rows) != 1 || !strings.Contains(warning, "old:") {
		t.Fatalf("partial result lost: %v, %q", rows, warning)
	}
	fail = true
	m.refresh(context.Background())
	rows, warning = m.Snapshot()
	if len(rows) != 0 || !strings.Contains(warning, "docker unavailable") {
		t.Fatalf("failure retained stale rows: %v, %q", rows, warning)
	}
	fail = false
	m.refresh(context.Background())
	rows, _ = m.Snapshot()
	if len(rows) != 1 {
		t.Fatal("monitor did not recover")
	}
}

func TestMonitorRejectsMalformedJSON(t *testing.T) {
	for _, data := range []string{"not JSON", `{"Name":`, `[{"Name":"bad"}] trailing`} {
		m := &Monitor{run: func(context.Context, ...string) ([]byte, error) {
			return []byte(data), nil
		}}
		m.refresh(context.Background())
		rows, warning := m.Snapshot()
		if len(rows) != 0 || warning == "" {
			t.Fatalf("malformed JSON accepted: %q", data)
		}
	}
}

func TestMonitorCancellation(t *testing.T) {
	started := make(chan struct{})
	done := make(chan struct{})
	m := &Monitor{run: func(ctx context.Context, args ...string) ([]byte, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { m.Run(ctx); close(done) }()
	<-started
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("monitor did not stop after cancellation")
	}
}

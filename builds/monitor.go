// Package builds discovers active BuildKit builds through the Docker Buildx CLI.
package builds

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

// Build represents a build, rather than the container hosting its builder.
type Build struct {
	Ref            string    `json:"ref"`
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	CompletedAt    time.Time `json:"completed_at"`
	TotalSteps     int       `json:"total_steps"`
	CompletedSteps int       `json:"completed_steps"`
	CachedSteps    int       `json:"cached_steps"`
	Builder        string
}

type commandRunner func(context.Context, ...string) ([]byte, error)

// Monitor polls outside the UI goroutine. Failed queries discard old rows so
// completed builds cannot remain displayed as running after a connection loss.
type Monitor struct {
	mu      sync.RWMutex
	builds  []Build
	warning string
	run     commandRunner
}

func NewMonitor() *Monitor { return &Monitor{run: dockerCommand} }

// Run refreshes immediately and then every two seconds until ctx is canceled.
func (m *Monitor) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		m.refresh(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (m *Monitor) Snapshot() ([]Build, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]Build(nil), m.builds...), m.warning
}

func dockerCommand(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

type builder struct {
	Name  string
	Nodes []struct{ Status string }
}

// decodeRecords accepts both newline-delimited JSON (Buildx's output) and arrays.
func decodeRecords[T any](data []byte) ([]T, error) {
	data = bytes.TrimSpace(data)
	var records []T
	if len(data) > 0 && data[0] == '[' {
		err := json.Unmarshal(data, &records)
		return records, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	for {
		var record T
		if err := dec.Decode(&record); err != nil {
			if err == io.EOF {
				return records, nil
			}
			return nil, err
		}
		records = append(records, record)
	}
}

func (m *Monitor) refresh(ctx context.Context) {
	rows, warning := m.query(ctx)
	m.mu.Lock()
	m.builds, m.warning = rows, warning
	m.mu.Unlock()
}

func (m *Monitor) query(ctx context.Context) ([]Build, string) {
	out, err := m.run(ctx, "buildx", "ls", "--format", "json")
	if err != nil {
		return nil, "Builds unavailable (requires Docker Buildx): " + err.Error()
	}
	builders, err := decodeRecords[builder](out)
	if err != nil {
		return nil, "Builds unavailable: invalid Buildx builder JSON"
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	var rows []Build
	var warnings []string
	seen := make(map[string]bool)
	// Limit simultaneous CLI processes for installations with many builders.
	slots := make(chan struct{}, 4)
	for _, b := range builders {
		if b.Name == "" || seen[b.Name] {
			continue
		}
		seen[b.Name] = true
		running := false
		for _, n := range b.Nodes {
			running = running || n.Status == "running"
		}
		if !running {
			continue
		}
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			case <-ctx.Done():
				return
			}
			out, err := m.run(ctx, "buildx", "--builder", name, "history", "ls", "--filter", "status=running", "--format", "json")
			var records []Build
			if err == nil {
				records, err = decodeRecords[Build](out)
			}
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("%s: %v", name, err))
				return
			}
			for _, r := range records {
				// Check locally too: never display completed history as active builds.
				if r.Ref != "" && strings.EqualFold(r.Status, "running") && r.CompletedAt.IsZero() {
					r.Builder = name
					rows = append(rows, r)
				}
			}
		}(b.Name)
	}
	wg.Wait()
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CreatedAt.Equal(rows[j].CreatedAt) {
			return rows[i].Ref < rows[j].Ref
		}
		return rows[i].CreatedAt.Before(rows[j].CreatedAt)
	})
	if len(warnings) > 0 {
		sort.Strings(warnings)
		return rows, "Build history unavailable for " + strings.Join(warnings, "; ")
	}
	return rows, ""
}

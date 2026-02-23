// Package telemetry provides observability for the AgentSpec runtime.
package telemetry

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Metrics collects Prometheus-style metrics for the AgentSpec runtime.
type Metrics struct {
	mu sync.RWMutex

	// Counters
	invocationsTotal map[string]int64 // key: agent,status
	tokensTotal      map[string]int64 // key: agent,type
	toolCallsTotal   map[string]int64 // key: agent,tool,status

	// Histograms (simplified: bucket counts + sum + count)
	invocationDurations map[string]*histogram // key: agent
}

type histogram struct {
	buckets []float64
	counts  []int64
	sum     float64
	count   int64
}

var defaultBuckets = []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60}

func newHistogram() *histogram {
	return &histogram{
		buckets: defaultBuckets,
		counts:  make([]int64, len(defaultBuckets)+1), // +1 for +Inf
	}
}

func (h *histogram) observe(value float64) {
	h.sum += value
	h.count++
	for i, b := range h.buckets {
		if value <= b {
			h.counts[i]++
		}
	}
	h.counts[len(h.buckets)]++ // +Inf always counts
}

// NewMetrics creates a new Metrics collector.
func NewMetrics() *Metrics {
	return &Metrics{
		invocationsTotal:    make(map[string]int64),
		tokensTotal:         make(map[string]int64),
		toolCallsTotal:      make(map[string]int64),
		invocationDurations: make(map[string]*histogram),
	}
}

// RecordInvocation records a completed agent invocation.
func (m *Metrics) RecordInvocation(agent, status string, duration time.Duration, inputTokens, outputTokens int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Increment invocation counter
	key := fmt.Sprintf("%s,%s", agent, status)
	m.invocationsTotal[key]++

	// Record duration
	h, ok := m.invocationDurations[agent]
	if !ok {
		h = newHistogram()
		m.invocationDurations[agent] = h
	}
	h.observe(duration.Seconds())

	// Record tokens
	m.tokensTotal[fmt.Sprintf("%s,input", agent)] += int64(inputTokens)
	m.tokensTotal[fmt.Sprintf("%s,output", agent)] += int64(outputTokens)
}

// RecordToolCall records a tool call.
func (m *Metrics) RecordToolCall(agent, tool, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s,%s,%s", agent, tool, status)
	m.toolCallsTotal[key]++
}

// Handler returns an HTTP handler that serves Prometheus-format metrics.
func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		m.mu.RLock()
		defer m.mu.RUnlock()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		var sb strings.Builder

		// Invocations counter
		sb.WriteString("# HELP agentspec_invocations_total Total agent invocations\n")
		sb.WriteString("# TYPE agentspec_invocations_total counter\n")
		for _, key := range sortedKeys(m.invocationsTotal) {
			parts := strings.SplitN(key, ",", 2)
			fmt.Fprintf(&sb, "agentspec_invocations_total{agent=%q,status=%q} %d\n",
				parts[0], parts[1], m.invocationsTotal[key])
		}
		sb.WriteString("\n")

		// Duration histogram
		sb.WriteString("# HELP agentspec_invocation_duration_seconds Invocation duration\n")
		sb.WriteString("# TYPE agentspec_invocation_duration_seconds histogram\n")
		for _, agent := range sortedMapKeys(m.invocationDurations) {
			h := m.invocationDurations[agent]
			cumulative := int64(0)
			for i, b := range h.buckets {
				cumulative += h.counts[i]
				fmt.Fprintf(&sb, "agentspec_invocation_duration_seconds_bucket{agent=%q,le=\"%.3g\"} %d\n",
					agent, b, cumulative)
			}
			cumulative += h.counts[len(h.buckets)]
			fmt.Fprintf(&sb, "agentspec_invocation_duration_seconds_bucket{agent=%q,le=\"+Inf\"} %d\n",
				agent, cumulative)
			fmt.Fprintf(&sb, "agentspec_invocation_duration_seconds_sum{agent=%q} %.6f\n",
				agent, h.sum)
			fmt.Fprintf(&sb, "agentspec_invocation_duration_seconds_count{agent=%q} %d\n",
				agent, h.count)
		}
		sb.WriteString("\n")

		// Tokens counter
		sb.WriteString("# HELP agentspec_tokens_total Tokens consumed\n")
		sb.WriteString("# TYPE agentspec_tokens_total counter\n")
		for _, key := range sortedKeys(m.tokensTotal) {
			parts := strings.SplitN(key, ",", 2)
			fmt.Fprintf(&sb, "agentspec_tokens_total{agent=%q,type=%q} %d\n",
				parts[0], parts[1], m.tokensTotal[key])
		}
		sb.WriteString("\n")

		// Tool calls counter
		sb.WriteString("# HELP agentspec_tool_calls_total Tool call count\n")
		sb.WriteString("# TYPE agentspec_tool_calls_total counter\n")
		for _, key := range sortedKeys(m.toolCallsTotal) {
			parts := strings.SplitN(key, ",", 3)
			fmt.Fprintf(&sb, "agentspec_tool_calls_total{agent=%q,tool=%q,status=%q} %d\n",
				parts[0], parts[1], parts[2], m.toolCallsTotal[key])
		}

		_, _ = w.Write([]byte(sb.String()))
	})
}

func sortedKeys(m map[string]int64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

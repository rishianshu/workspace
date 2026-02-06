// Package agentengine provides lightweight tracing for agent runs.
package agentengine

import "time"

// Trace captures lifecycle events for a single run.
type Trace struct {
	ID      string
	Started time.Time
	Events  []TraceEvent
}

// TraceEvent represents a single event in a trace.
type TraceEvent struct {
	Name   string
	Detail string
	At     time.Time
}

// NewTrace creates a new trace.
func NewTrace(id string) *Trace {
	return &Trace{
		ID:      id,
		Started: time.Now(),
		Events:  make([]TraceEvent, 0, 8),
	}
}

// AddEvent appends an event to the trace.
func (t *Trace) AddEvent(name, detail string) {
	if t == nil {
		return
	}
	t.Events = append(t.Events, TraceEvent{
		Name:   name,
		Detail: detail,
		At:     time.Now(),
	})
}

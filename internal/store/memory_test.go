package store

import (
	"testing"

	"github.com/brainproxy/brainproxy/internal/models"
)

func TestMemoryStore_Add(t *testing.T) {
	s := NewMemoryStore(3)

	s.Add(&models.RequestEvent{ID: "1"})
	s.Add(&models.RequestEvent{ID: "2"})
	s.Add(&models.RequestEvent{ID: "3"})

	events := s.List()
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
}

func TestMemoryStore_RingBufferEviction(t *testing.T) {
	s := NewMemoryStore(3)

	s.Add(&models.RequestEvent{ID: "1"})
	s.Add(&models.RequestEvent{ID: "2"})
	s.Add(&models.RequestEvent{ID: "3"})
	s.Add(&models.RequestEvent{ID: "4"}) // should evict "1"

	events := s.List()
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].ID != "2" {
		t.Errorf("expected oldest event ID '2', got '%s'", events[0].ID)
	}
	if events[2].ID != "4" {
		t.Errorf("expected newest event ID '4', got '%s'", events[2].ID)
	}
}

func TestMemoryStore_Get(t *testing.T) {
	s := NewMemoryStore(10)
	s.Add(&models.RequestEvent{ID: "abc"})

	event := s.Get("abc")
	if event == nil {
		t.Fatal("expected to find event 'abc'")
	}
	if event.ID != "abc" {
		t.Errorf("expected ID 'abc', got '%s'", event.ID)
	}

	missing := s.Get("nonexistent")
	if missing != nil {
		t.Error("expected nil for nonexistent event")
	}
}

func TestMemoryStore_Update(t *testing.T) {
	s := NewMemoryStore(10)
	s.Add(&models.RequestEvent{ID: "1"})

	updated := s.Update("1", func(e *models.RequestEvent) {
		e.Analysis = &models.Analysis{Model: "claude-3"}
	})
	if !updated {
		t.Fatal("expected update to succeed")
	}

	event := s.Get("1")
	if event.Analysis == nil || event.Analysis.Model != "claude-3" {
		t.Error("expected analysis to be updated")
	}

	notFound := s.Update("nonexistent", func(e *models.RequestEvent) {})
	if notFound {
		t.Error("expected update to fail for nonexistent event")
	}
}

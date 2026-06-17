package store

import (
	"sync"

	"github.com/brainproxy/brainproxy/internal/models"
)

// MemoryStore is a thread-safe ring buffer for request events.
type MemoryStore struct {
	mu       sync.RWMutex
	buffer   []*models.RequestEvent
	capacity int
	head     int
	count    int
}

func NewMemoryStore(capacity int) *MemoryStore {
	return &MemoryStore{
		buffer:   make([]*models.RequestEvent, capacity),
		capacity: capacity,
	}
}

func (s *MemoryStore) Add(event *models.RequestEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buffer[s.head] = event
	s.head = (s.head + 1) % s.capacity
	if s.count < s.capacity {
		s.count++
	}
}

func (s *MemoryStore) Get(id string) *models.RequestEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, event := range s.buffer {
		if event != nil && event.ID == id {
			return event
		}
	}
	return nil
}

func (s *MemoryStore) List() []*models.RequestEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*models.RequestEvent, 0, s.count)
	if s.count < s.capacity {
		for i := 0; i < s.count; i++ {
			result = append(result, s.buffer[i])
		}
	} else {
		for i := 0; i < s.capacity; i++ {
			idx := (s.head + i) % s.capacity
			result = append(result, s.buffer[idx])
		}
	}
	return result
}

func (s *MemoryStore) Update(id string, fn func(e *models.RequestEvent)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, event := range s.buffer {
		if event != nil && event.ID == id {
			fn(event)
			return true
		}
	}
	return false
}

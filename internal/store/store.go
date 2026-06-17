package store

import "github.com/brainproxy/brainproxy/internal/models"

// Store defines the interface for event storage.
type Store interface {
	Add(event *models.RequestEvent)
	Get(id string) *models.RequestEvent
	List() []*models.RequestEvent
	Update(id string, fn func(e *models.RequestEvent)) bool
}

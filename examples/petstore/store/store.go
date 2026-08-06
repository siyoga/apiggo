// Package store is a tiny in-memory pet repository. It stands in for whatever
// real dependency (a database, another service) a handler would talk to, and is
// injected into the generated handler stubs via their New constructors.
package store

import (
	"sync"

	dto "github.com/siyoga/apiggo/examples/petstore/generated/dto"
)

// Store keeps pets in memory, keyed by id. It is safe for concurrent use.
type Store struct {
	mu   sync.RWMutex
	pets map[int64]dto.Pet
	next int64
}

// New returns an empty Store.
func New() *Store {
	return &Store{pets: make(map[int64]dto.Pet), next: 1}
}

// Get returns the pet with the given id, or ok=false if none exists.
func (s *Store) Get(id int64) (dto.Pet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pet, ok := s.pets[id]
	return pet, ok
}

// Create assigns the next id to pet, stores it, and returns the stored copy.
func (s *Store) Create(pet dto.Pet) dto.Pet {
	s.mu.Lock()
	defer s.mu.Unlock()
	pet.Id = s.next
	s.next++
	s.pets[pet.Id] = pet
	return pet
}

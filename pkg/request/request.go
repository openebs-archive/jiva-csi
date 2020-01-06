package request

import (
	"sync"
)

// Interface is used to handle the inflight requests
type Interface interface {
	String() string
}

// Transition is a struct used to manage inflight volume creation request.
type Transition struct {
	mux    *sync.Mutex
	volume map[string]bool
}

// NewTransition instanciates Transition.
func NewTransition() *Transition {
	return &Transition{
		mux:    &sync.Mutex{},
		volume: make(map[string]bool),
	}
}

// Insert insert the volume create req hash into map
// TODO: Add request info as well to know about which request is in progress
func (t *Transition) Insert(volID string) bool {
	t.mux.Lock()
	defer t.mux.Unlock()

	_, ok := t.volume[volID]
	if ok {
		return false
	}

	t.volume[volID] = true
	return true
}

// Delete removes the req from the map
func (t *Transition) Delete(volID string) {
	t.mux.Lock()
	defer t.mux.Unlock()

	delete(t.volume, volID)
}

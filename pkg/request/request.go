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
func (t *Transition) Insert(req Interface) bool {
	t.mux.Lock()
	defer t.mux.Unlock()

	hash := req.String()

	_, ok := t.volume[hash]
	if ok {
		return false
	}

	t.volume[hash] = true
	return true
}

// Delete removes the req from the map
func (t *Transition) Delete(req Interface) {
	t.mux.Lock()
	defer t.mux.Unlock()

	delete(t.volume, req.String())
}

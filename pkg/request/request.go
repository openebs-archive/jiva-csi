package request

import (
	"sync"
)

// Transition is a struct used to manage inflight volume creation request.
type Transition struct {
	mux    *sync.Mutex
	volume map[string]string
}

// NewTransition instanciates Transition.
func NewTransition() *Transition {
	return &Transition{
		mux:    &sync.Mutex{},
		volume: make(map[string]string),
	}
}

func (t *Transition) GetOperation(volID string) string {
	t.mux.Lock()
	defer t.mux.Unlock()

	_, ok := t.volume[volID]
	if ok {
		return t.volume[volID]
	}
	return ""
}

// Insert insert the volume create req hash into map
func (t *Transition) Insert(volID string, ops string) bool {
	t.mux.Lock()
	defer t.mux.Unlock()

	_, ok := t.volume[volID]
	if ok {
		return false
	}

	t.volume[volID] = ops
	return true
}

// Delete removes the req from the map
func (t *Transition) Delete(volID string) {
	t.mux.Lock()
	defer t.mux.Unlock()

	delete(t.volume, volID)
}

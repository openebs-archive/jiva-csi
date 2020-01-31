package request

import (
	"fmt"
	"sync"
)

var (

	// TransitionVolList contains the list of volumes under transition
	// This list is protected by TransitionVolListLock
	TransitionVolList map[string]string

	// TransitionVolListLock is required to protect the above Volumes list
	TransitionVolListLock sync.RWMutex
)

func init() {
	TransitionVolList = make(map[string]string)
}

func RemoveVolumeFromTransitionList(volumeID string) {
	TransitionVolListLock.Lock()
	defer TransitionVolListLock.Unlock()
	delete(TransitionVolList, volumeID)
}

func AddVolumeToTransitionList(volumeID string, req string) error {
	TransitionVolListLock.Lock()
	defer TransitionVolListLock.Unlock()

	if _, ok := TransitionVolList[volumeID]; ok {
		return fmt.Errorf("Volume Busy, %v is already in progress",
			TransitionVolList[volumeID])
	}
	TransitionVolList[volumeID] = req
	return nil
}

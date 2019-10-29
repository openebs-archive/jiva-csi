package driver

import (
	"k8s.io/kubernetes/pkg/util/mount"
)

type NodeMounter struct {
	mount.SafeFormatAndMount
}

func newNodeMounter() *NodeMounter {
	return &NodeMounter{
		mount.SafeFormatAndMount{
			Interface: mount.New(""),
			Exec:      mount.NewOsExec(),
		},
	}
}

func (m *NodeMounter) GetDeviceName(mountPath string) (string, int, error) {
	return mount.GetDeviceNameFromMount(m, mountPath)
}

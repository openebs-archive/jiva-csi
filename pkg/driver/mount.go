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

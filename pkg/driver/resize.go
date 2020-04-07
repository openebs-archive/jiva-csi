package driver

import (
	"github.com/sirupsen/logrus"
	"k8s.io/kubernetes/pkg/util/mount"
)

type resizeInput struct {
	volumePath   string
	fsType       string
	iqn          string
	targetPortal string
	exec         mount.Exec
}

func (r resizeInput) volume(list []mount.MountPoint) error {
	for _, mpt := range list {
		if mpt.Path == r.volumePath {
			err := r.reScan()
			if err != nil {
				return err
			}
			switch r.fsType {
			case "ext4":
				err = r.resizeExt4(mpt.Device)
			case "xfs":
				err = r.resizeXFS(r.volumePath)
			}
			if err != nil {
				return err
			}
			break
		}
	}
	return nil
}

// ReScan rescans all the iSCSI sessions on the host
func (r resizeInput) reScan() error {
	logrus.Info("Rescan ISCSI session")
	out, err := r.exec.Run("iscsiadm", "-m", "node", "-T", r.iqn, "-P", r.targetPortal, "--rescan")
	if err != nil {
		logrus.Errorf("iscsi: rescan failed error: %s", string(out))
		return err
	}
	return nil
}

// ResizeExt4 can be used to run a resize command on the ext4 filesystem
// to expand the filesystem to the actual size of the device
func (r resizeInput) resizeExt4(path string) error {
	out, err := r.exec.Run("resize2fs", path)
	if err != nil {
		logrus.Errorf("iscsi: resize failed error: %s", string(out))
		return err
	}
	return nil
}

// ResizeXFS can be used to run a resize command on the xfs filesystem
// to expand the filesystem to the actual size of the device
func (r resizeInput) resizeXFS(path string) error {
	out, err := r.exec.Run("xfs_growfs", path)
	if err != nil {
		logrus.Errorf("iscsi: resize failed error: %s", string(out))
		return err
	}
	return nil
}

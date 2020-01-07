package driver

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/sys/unix"
)

func getStatistics(volumePath string) ([]*csi.VolumeUsage, error) {
	var statfs unix.Statfs_t
	// See http://man7.org/linux/man-pages/man2/statfs.2.html for details.
	err := unix.Statfs(volumePath, &statfs)
	if err != nil {
		return nil, err
	}

	inBytes := csi.VolumeUsage{
		Available: int64(statfs.Bavail) * int64(statfs.Bsize),
		Total:     int64(statfs.Blocks) * int64(statfs.Bsize),
		Used:      (int64(statfs.Blocks) - int64(statfs.Bfree)) * int64(statfs.Bsize),
		Unit:      csi.VolumeUsage_BYTES,
	}

	inInodes := csi.VolumeUsage{
		Available: int64(statfs.Ffree),
		Total:     int64(statfs.Files),
		Used:      int64(statfs.Files) - int64(statfs.Ffree),
		Unit:      csi.VolumeUsage_INODES,
	}

	volStats := []*csi.VolumeUsage{
		&inBytes,
		&inInodes,
	}
	return volStats, nil
}

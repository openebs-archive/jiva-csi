/*
Copyright © 2019 The OpenEBS Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-lib-iscsi/iscsi"
	"github.com/openebs/jiva-csi/pkg/kubernetes/client"
	jv "github.com/openebs/jiva-operator/pkg/apis/openebs/v1alpha1"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// FSTypeExt2 represents the ext2 filesystem type
	FSTypeExt2 = "ext2"
	// FSTypeExt3 represents the ext3 filesystem type
	FSTypeExt3 = "ext3"
	// FSTypeExt4 represents the ext4 filesystem type
	FSTypeExt4 = "ext4"
	// FSTypeXfs represents te xfs filesystem type
	FSTypeXfs = "xfs"

	defaultFsType = FSTypeExt4

	MaxRetryCount = 10
)

var (
	ValidFSTypes = []string{FSTypeExt2, FSTypeExt3, FSTypeExt4, FSTypeXfs}
)

var (
	// nodeCaps represents the capability of node service.
	nodeCaps = []csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
	}
)

// node is the server implementation
// for CSI NodeServer
type node struct {
	client  *client.Client
	driver  *CSIDriver
	mounter *NodeMounter
}

// NewNode returns a new instance
// of CSI NodeServer
func NewNode(d *CSIDriver, cli *client.Client) csi.NodeServer {
	return &node{
		client: cli,
		driver: d,
	}
}

func (ns *node) attachDisk(instance *jv.JivaVolume) (string, error) {
	connector := iscsi.Connector{
		VolumeName:    instance.Name,
		TargetIqn:     instance.Spec.ISCSISpec.Iqn,
		Port:          fmt.Sprint(instance.Spec.ISCSISpec.TargetPort),
		Lun:           instance.Spec.ISCSISpec.Lun,
		Interface:     instance.Spec.ISCSISpec.ISCSIInterface,
		TargetPortals: instance.Spec.ISCSISpec.TargetPortals,
	}
	devicePath, err := iscsi.Connect(connector)
	if err != nil {
		return "", err
	}

	if devicePath == "" {
		return "", fmt.Errorf("connect reported success, but no path returned")
	}
	return devicePath, err
}

func (ns *node) waitForVolumeToBeReady(volID string) (*jv.JivaVolume, error) {
	var retry int32
	for {
		if err := ns.client.Set(); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		instance, err := ns.client.GetJivaVolume(volID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		time.Sleep(2 * time.Second)
		retry++
		if instance.Status.Phase == jv.JivaVolumePhaseCreated {
			return instance, nil
		} else if retry > MaxRetryCount {
			if instance.Status.Status == "RO" {
				status := instance.Status.ReplicaStatuses
				if status != nil {
					return nil, fmt.Errorf("Volume is RO: replica status: %+v", status)
				}
				return nil, fmt.Errorf("Volume is not ready: replicas may not be connected")
			}
			return nil, fmt.Errorf("Volume is not ready: volume status is unknown")
		}
	}
}

func (ns *node) waitForVolumeToBeReachable(targetIP string) error {
	var (
		retries int
		err     error
		conn    net.Conn
	)

	for {
		// Create a connection to test if the iSCSI Portal is reachable,
		if conn, err = net.Dial("tcp", targetIP); err == nil {
			conn.Close()
			logrus.Debug("Volume is reachable to create connections")
			return nil
		}
		// wait until the iSCSI targetPortal is reachable
		// There is no pointn of triggering iSCSIadm login commands
		// until the portal is reachable
		time.Sleep(2 * time.Second)
		retries++
		if retries >= MaxRetryCount {
			// Let the caller function decide further if the volume is
			// not reachable even after 12 seconds ( This number was arrived at
			// based on the kubelets retrying logic. Kubelet retries to publish
			// volume after every 14s )
			return fmt.Errorf(
				"iSCSI Target not reachable, TargetPortal %v, err:%v",
				targetIP, err)
		}
	}
}

// NodeStageVolume mounts the volume on the staging
// path
//
// This implements csi.NodeServer
func (ns *node) NodeStageVolume(
	ctx context.Context,
	req *csi.NodeStageVolumeRequest,
) (*csi.NodeStageVolumeResponse, error) {

	logrus.Infof("NodeStageVolume: called with args %+v", *req)

	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	volCap := req.GetVolumeCapability()
	if volCap == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not provided")
	}

	if !isValidVolumeCapabilities([]*csi.VolumeCapability{volCap}) {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not supported")
	}

	mount := volCap.GetMount()
	if mount == nil {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume: mount is nil within volume capability")
	}

	fsType := mount.GetFsType()
	if len(fsType) == 0 {
		fsType = defaultFsType
	}

	stagingPath := req.GetStagingTargetPath()
	if len(stagingPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "staging path is empty")
	}
	// Check if volume is ready to serve IOs,
	// info is fetched from the JivaVolume CR
	logrus.Info("NodeStageVolume: wait for the volume to be ready")
	instance, err := ns.waitForVolumeToBeReady(volumeID)
	if err != nil {
		return nil, err
	}

	// A temporary TCP connection is made to the volume to check if its
	// reachable
	logrus.Info("NodeStageVolume: wait for the iscsi target to be ready")
	if err := ns.waitForVolumeToBeReachable(
		instance.Spec.ISCSISpec.TargetIP,
	); err != nil {
		return nil,
			status.Error(codes.Internal, err.Error())
	}

	devicePath, err := ns.attachDisk(instance)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	instance.Spec.MountInfo.FSType = fsType
	instance.Spec.MountInfo.DevicePath = devicePath
	instance.Spec.MountInfo.Path = stagingPath
	if err := ns.client.UpdateJivaVolume(instance); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := ns.formatAndMount(req, instance.Spec.MountInfo.DevicePath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

// NodeUnstageVolume unmounts the volume from
// the staging path
//
// This implements csi.NodeServer
func (ns *node) NodeUnstageVolume(
	ctx context.Context,
	req *csi.NodeUnstageVolumeRequest,
) (*csi.NodeUnstageVolumeResponse, error) {

	volID := req.GetVolumeId()
	if volID == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeUnstageVolume Volume ID must be provided")
	}

	if err := ns.client.Set(); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	target := req.GetStagingTargetPath()
	if len(target) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Staging target not provided")
	}

	// Check if target directory is a mount point. GetDeviceNameFromMount
	// given a mnt point, finds the device from /proc/mounts
	// returns the device name, reference count, and error code
	dev, refCount, err := ns.mounter.GetDeviceName(target)
	if err != nil {
		msg := fmt.Sprintf("failed to check if volume is mounted: %v", err)
		return nil, status.Error(codes.Internal, msg)
	}

	// From the spec: If the volume corresponding to the volume_id
	// is not staged to the staging_target_path, the Plugin MUST
	// reply 0 OK.
	if refCount == 0 {
		logrus.Infof("NodeUnstageVolume: %s target not mounted", target)
		return &csi.NodeUnstageVolumeResponse{}, nil
	}

	if refCount > 1 {
		logrus.Warningf("NodeUnstageVolume: found %d references to device %s mounted at target path %s", refCount, dev, target)
	}

	logrus.Infof("NodeUnstageVolume: unmounting %s", target)
	err = ns.mounter.Unmount(target)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmount target %q: %v", target, err)
	}

	instance, err := ns.client.GetJivaVolume(volID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := iscsi.Disconnect(instance.Spec.ISCSISpec.Iqn, instance.Spec.ISCSISpec.TargetPortals); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := os.RemoveAll(instance.Spec.MountInfo.Path); err != nil {
		logrus.Errorf("Failed to remove mount path, err: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	logrus.Infof("NodeUnstageVolume: detaching device %v is finished", instance.Spec.MountInfo.DevicePath)

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (ns *node) formatAndMount(req *csi.NodeStageVolumeRequest, devicePath string) error {
	// Mount device
	mntPath := req.GetStagingTargetPath()
	notMnt, err := ns.mounter.IsLikelyNotMountPoint(mntPath)
	if err != nil && !os.IsNotExist(err) {
		if err := os.MkdirAll(mntPath, 0750); err != nil {
			logrus.Errorf("iscsi: failed to mkdir %s, error", mntPath)
			return err
		}
	}

	if !notMnt {
		logrus.Infof("Volume %s has been mounted already at %v", req.GetVolumeId(), mntPath)
		return nil
	}

	fsType := req.GetVolumeCapability().GetMount().GetFsType()
	options := []string{}
	mountFlags := req.GetVolumeCapability().GetMount().GetMountFlags()
	options = append(options, mountFlags...)

	err = ns.mounter.FormatAndMount(devicePath, mntPath, fsType, options)
	if err != nil {
		logrus.Errorf(
			"Failed to mount iscsi volume %s [%s, %s] to %s, error %v",
			req.GetVolumeId(), devicePath, fsType, mntPath, err,
		)
		return err
	}
	return nil
}

// NodePublishVolume publishes (mounts) the volume
// at the corresponding node at a given path
//
// This implements csi.NodeServer
func (ns *node) NodePublishVolume(
	ctx context.Context,
	req *csi.NodePublishVolumeRequest,
) (*csi.NodePublishVolumeResponse, error) {

	logrus.Infof("NodePublishVolume: called with args %+v", *req)
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "Volume ID not provided")
	}

	target := req.GetTargetPath()
	if len(target) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "Target path not provided")
	}

	volCap := req.GetVolumeCapability()
	if volCap == nil {
		return nil, status.Error(codes.FailedPrecondition, "Volume capability not provided")
	}

	if !isValidVolumeCapabilities([]*csi.VolumeCapability{volCap}) {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not supported")
	}

	if err := ns.client.Set(); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	mountOptions := []string{"bind"}
	if req.GetReadonly() {
		mountOptions = append(mountOptions, "ro")
	}
	switch mode := volCap.GetAccessType().(type) {
	case *csi.VolumeCapability_Block:
		return &csi.NodePublishVolumeResponse{}, status.Error(codes.Unimplemented, "doesn't support block device provisioning")
	case *csi.VolumeCapability_Mount:
		if err := ns.nodePublishVolumeForFileSystem(req, mountOptions, mode); err != nil {
			return nil, err
		}
	}
	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *node) nodePublishVolumeForFileSystem(req *csi.NodePublishVolumeRequest, mountOptions []string, mode *csi.VolumeCapability_Mount) error {
	target := req.GetTargetPath()
	source := req.GetStagingTargetPath()
	if m := mode.Mount; m != nil {
		hasOption := func(options []string, opt string) bool {
			for _, o := range options {
				if o == opt {
					return true
				}
			}
			return false
		}
		for _, f := range m.MountFlags {
			if !hasOption(mountOptions, f) {
				mountOptions = append(mountOptions, f)
			}
		}
	}

	logrus.Infof("NodePublishVolume: creating dir %s", target)
	if err := ns.mounter.MakeDir(target); err != nil {
		return status.Errorf(codes.Internal, "Could not create dir %q: %v", target, err)
	}

	fsType := mode.Mount.GetFsType()
	if len(fsType) == 0 {
		fsType = defaultFsType
	}

	logrus.Infof("NodePublishVolume: mounting %s at %s with option %s as fstype %s", source, target, mountOptions, fsType)
	if err := ns.mounter.Mount(source, target, fsType, mountOptions); err != nil {
		if removeErr := os.Remove(target); removeErr != nil {
			return status.Errorf(codes.Internal, "Could not remove mount target %q: %v", target, err)
		}
		return status.Errorf(codes.Internal, "Could not mount %q at %q: %v", source, target, err)
	}

	return nil
}

// NodeUnpublishVolume unpublishes (unmounts) the volume
// from the corresponding node from the given path
//
// This implements csi.NodeServer
func (ns *node) NodeUnpublishVolume(
	ctx context.Context,
	req *csi.NodeUnpublishVolumeRequest,
) (*csi.NodeUnpublishVolumeResponse, error) {

	logrus.Infof("NodeUnpublishVolume: called with args %+v", *req)
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	target := req.GetTargetPath()
	if len(target) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path not provided")
	}

	notMnt, err := ns.mounter.IsLikelyNotMountPoint(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, status.Error(codes.NotFound, "targetpath not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !notMnt {
		return nil, status.Error(codes.Internal, "Volume not mounted")
	}

	logrus.Infof("NodeUnpublishVolume: unmounting %s", target)
	if err := ns.mounter.Unmount(target); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmount %q: %v", target, err)
	}
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeGetInfo returns node details
//
// This implements csi.NodeServer
func (ns *node) NodeGetInfo(
	ctx context.Context,
	req *csi.NodeGetInfoRequest,
) (*csi.NodeGetInfoResponse, error) {

	return &csi.NodeGetInfoResponse{
		NodeId: ns.driver.config.NodeID,
	}, nil
}

// NodeGetCapabilities returns capabilities supported
// by this node service
//
// This implements csi.NodeServer
func (ns *node) NodeGetCapabilities(
	ctx context.Context,
	req *csi.NodeGetCapabilitiesRequest,
) (*csi.NodeGetCapabilitiesResponse, error) {

	logrus.Infof("NodeGetCapabilities: called with args %+v", *req)
	var caps []*csi.NodeServiceCapability
	for _, cap := range nodeCaps {
		c := &csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: cap,
				},
			},
		}
		caps = append(caps, c)
	}
	return &csi.NodeGetCapabilitiesResponse{Capabilities: caps}, nil
}

// TODO
// Verify if this needs to be implemented
//
// NodeExpandVolume resizes the filesystem if required
//
// If ControllerExpandVolumeResponse returns true in
// node_expansion_required then FileSystemResizePending
// condition will be added to PVC and NodeExpandVolume
// operation will be queued on kubelet
//
// This implements csi.NodeServer
func (ns *node) NodeExpandVolume(
	ctx context.Context,
	req *csi.NodeExpandVolumeRequest,
) (*csi.NodeExpandVolumeResponse, error) {

	return nil, nil
}

// NodeGetVolumeStats returns statistics for the
// given volume
//
// This implements csi.NodeServer
func (ns *node) NodeGetVolumeStats(
	ctx context.Context,
	in *csi.NodeGetVolumeStatsRequest,
) (*csi.NodeGetVolumeStatsResponse, error) {

	return nil, status.Error(codes.Unimplemented, "")
}

func (ns *node) validateNodePublishReq(
	req *csi.NodePublishVolumeRequest,
) error {
	if req.GetVolumeCapability() == nil {
		return status.Error(codes.InvalidArgument,
			"Volume capability missing in request")
	}

	if len(req.GetVolumeId()) == 0 {
		return status.Error(codes.InvalidArgument,
			"Volume ID missing in request")
	}
	return nil
}

func (ns *node) validateNodeUnpublishReq(
	req *csi.NodeUnpublishVolumeRequest,
) error {
	if req.GetVolumeId() == "" {
		return status.Error(codes.InvalidArgument,
			"Volume ID missing in request")
	}

	if req.GetTargetPath() == "" {
		return status.Error(codes.InvalidArgument,
			"Target path missing in request")
	}
	return nil
}
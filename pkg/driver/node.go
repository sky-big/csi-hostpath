/*
Copyright 2019 The Kubernetes Authors.

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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	k8smount "k8s.io/utils/mount"

	"github.com/container-storage-interface/spec/lib/go/csi"
	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
)

const (
	CSIHostPathKey = "csi-hostpath-path"
	NsenterCmd     = "/nsenter --mount=/proc/1/ns/mnt"
)

type nodeServer struct {
	driver *HostPathCSIDriver
	*csicommon.DefaultNodeServer
	nodeID     string
	client     kubernetes.Interface
	k8smounter k8smount.Interface
}

// NewNodeServer create a NodeServer object
func NewNodeServer(d *HostPathCSIDriver, nodeID string) csi.NodeServer {
	var masterURL, kubeconfig string
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	return &nodeServer{
		driver:            d,
		DefaultNodeServer: csicommon.NewDefaultNodeServer(d.csiDriver),
		nodeID:            nodeID,
		k8smounter:        k8smount.New(""),
		client:            kubeClient,
	}
}

func (ns *nodeServer) GetNodeID() string {
	return ns.nodeID
}

func (ns *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	klog.Infof("NodeServer:NodePublishVolume Request :: %+v", *req)

	// parse request args.
	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.Internal, "targetPath is empty")
	}

	isMnt, err := ns.IsMounted(targetPath)
	if err != nil {
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			if err := os.MkdirAll(targetPath, 0750); err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			isMnt = false
		} else {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	devicePath, err := ns.getHostPath(ctx, req.GetVolumeId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !isMnt {
		err = ns.Mount(devicePath, targetPath)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		klog.Infof("NodeServer:NodePublishVolume Success :: mount successful devicePath = %s, targetPath = %s",
			devicePath, targetPath)
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	klog.Infof("NodeServer:NodeUnpublishVolume Request :: %+v", *req)

	targetPath := req.GetTargetPath()
	isMnt, err := ns.IsMounted(targetPath)
	if err != nil {
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			return nil, status.Error(codes.NotFound, "TargetPath not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !isMnt {
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}

	err = ns.Unmount(req.GetTargetPath())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	klog.Infof("NodeServer:NodeUnpublishVolume umount success :: volume = %s, targetPath = %s",
		req.GetVolumeId(), req.GetTargetPath())
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	klog.Infof("NodeServer:NodeUnstageVolume Request :: %+v", *req)
	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (ns *nodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	klog.Infof("NodeServer:NodeStageVolume Request :: %+v", *req)
	return &csi.NodeStageVolumeResponse{}, nil
}

func (ns *nodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	klog.Infof("NodeServer:NodeGetCapabilities Request :: %+v", *req)
	// currently there is a single NodeServer capability according to the spec
	nscap := &csi.NodeServiceCapability{
		Type: &csi.NodeServiceCapability_Rpc{
			Rpc: &csi.NodeServiceCapability_RPC{
				Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
			},
		},
	}
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			nscap,
		},
	}, nil
}

func (ns *nodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	klog.Infof("NodeServer:NodeExpandVolume Request :: %+v", *req)
	return &csi.NodeExpandVolumeResponse{}, nil
}

func (ns *nodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	klog.Infof("NodeServer:NodeGetInfo Request :: %+v", *req)
	return &csi.NodeGetInfoResponse{
		NodeId: ns.nodeID,
		// make sure that the driver works on this particular node only
		AccessibleTopology: &csi.Topology{
			Segments: map[string]string{
				TopologyNodeKey: ns.nodeID,
			},
		},
	}, nil
}

func (ns *nodeServer) getHostPath(ctx context.Context, volumeId string) (string, error) {
	pv, err := ns.client.CoreV1().PersistentVolumes().Get(ctx, volumeId, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("getHostpath Get PV(%s) Error: %s", volumeId, err.Error())
		return "", status.Error(codes.Internal, err.Error())
	}

	pvc, err := ns.client.CoreV1().PersistentVolumeClaims(pv.Spec.ClaimRef.Namespace).Get(ctx, pv.Spec.ClaimRef.Name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("getHostpath Get PVC(namespace = %s, name = %s) Error: %s",
			pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name, err.Error())
		return "", status.Error(codes.Internal, err.Error())
	}

	if hostPath, ok := pvc.Annotations[CSIHostPathKey]; ok {
		return hostPath, nil
	}
	return "", status.Error(codes.Internal, "pvc annotations csi hostpath not set")
}

func (ns *nodeServer) Mount(source, target string) error {
	if source == "" {
		return errors.New("source is not specified for mounting the volume")
	}

	if target == "" {
		return errors.New("target is not specified for mounting the volume")
	}

	mountCmd := []string{NsenterCmd, "mount"}
	mountCmd = append(mountCmd, "--bind")
	mountCmd = append(mountCmd, source)
	mountCmd = append(mountCmd, target)

	// create target, os.Mkdirall is noop if it exists
	err := os.MkdirAll(target, 0750)
	if err != nil {
		return err
	}

	klog.Infof("Mount %s to %s, the command is %v", source, target, mountCmd)

	out, err := exec.Command("sh", "-c", strings.Join(mountCmd, " ")).CombinedOutput()
	if err != nil {
		return fmt.Errorf("mounting failed: %v cmd: '%s %s' output: %q",
			err, mountCmd, strings.Join(mountCmd, " "), string(out))
	}

	return nil
}

func (ns *nodeServer) Unmount(target string) error {
	if target == "" {
		return errors.New("target is not specified for unmounting the volume")
	}

	umountCmd := []string{NsenterCmd, "umount"}
	umountCmd = append(umountCmd, target)

	klog.Infof("Unmount %s, the command is %v", target, umountCmd)

	out, err := exec.Command("sh", "-c", strings.Join(umountCmd, " ")).CombinedOutput()
	if err != nil {
		return fmt.Errorf("unmounting failed: %v cmd: '%s %s' output: %q",
			err, umountCmd, target, string(out))
	}

	return nil
}

func (ns *nodeServer) IsMounted(target string) (bool, error) {
	if target == "" {
		return false, errors.New("target is not specified for checking the mount")
	}

	findMntCmd := []string{NsenterCmd, "grep"}
	findMntCmd = append(findMntCmd, target)
	findMntCmd = append(findMntCmd, "proc/mounts")
	out, err := exec.Command("sh", "-c", strings.Join(findMntCmd, " ")).CombinedOutput()
	outStr := strings.TrimSpace(string(out))
	if err != nil {
		if outStr == "" {
			return false, nil
		}
		return false, fmt.Errorf("checking mounted failed: %v cmd: %q output: %q",
			err, findMntCmd, outStr)
	}
	if strings.Contains(outStr, target) {
		return true, nil
	}
	return false, nil
}

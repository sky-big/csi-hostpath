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
	"k8s.io/klog"

	"github.com/container-storage-interface/spec/lib/go/csi"
	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
)

const (
	DriverName    = "hostpath.csi.kubernetes.io"
	DriverVersion = "0.0.1"

	TopologyNodeKey = "topology.hostpath.csi/hostname"
)

type HostPathCSIDriver struct {
	driverName       string
	driverVersion    string
	nodeID           string
	csiDriver        *csicommon.CSIDriver
	endpoint         string
	idServer         *identityServer
	nodeServer       csi.NodeServer
	controllerServer *controllerServer
}

// NewHostPathCSIDriver create the identity/node/controller server and disk driver
func NewHostPathCSIDriver(driverName, driverVersion, nodeID, endpoint string) *HostPathCSIDriver {
	driver := &HostPathCSIDriver{}
	driver.driverName = driverName
	driver.driverVersion = driverVersion
	driver.nodeID = nodeID
	driver.endpoint = endpoint

	csiDriver := csicommon.NewCSIDriver(driverName, driverVersion, nodeID)
	driver.csiDriver = csiDriver
	driver.csiDriver.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
	})
	driver.csiDriver.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER})

	// Create GRPC servers
	driver.idServer = newIdentityServer(driver)
	driver.nodeServer = NewNodeServer(driver, nodeID)
	driver.controllerServer = newControllerServer(driver)

	return driver
}

func (d *HostPathCSIDriver) Run() {
	klog.Infof("HostPath CSI Driver(%s) version(%s) starting on node(%s) listen endpoint(%s)",
		d.driverName, d.driverVersion, d.nodeID, d.endpoint)

	server := csicommon.NewNonBlockingGRPCServer()
	server.Start(d.endpoint, d.idServer, d.controllerServer, d.nodeServer)
	server.Wait()
}

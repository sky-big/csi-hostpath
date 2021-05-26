# Kubernetes HostPath CSI Plugin

## Overview

HostPath CSI plugin implement an interface between CSI enabled Container Orchestrator, This is exactly the same as the native HostPath plugin.
This project purpose is to learn how to write CSI plugin.

## Quick Start

### Deploy Plugin

1. Clone the project on your Kubernetes cluster master node:
```
$ git clone https://github.com/sky-big/csi-hostpath.git
$ cd csi-hostpath
```

2. To deploy the Plugin on your Kubernetes cluster, need ` Helm `:
```
$ helm install csi-hostpath ./deploy
```

3. Use command ```kubectl get pods -n kube-system```to check Plugin status like:
```
NAMESPACE                    NAME                                                     READY   STATUS             RESTARTS   AGE
kube-system                  hostpath-csi-controller-0                                1/1     Running            0          11m
kube-system                  hostpath-csi-node-97hzx                                  2/2     Running            0          11m
kube-system                  hostpath-csi-node-hdv6n                                  2/2     Running            0          11m
kube-system                  hostpath-csi-node-z4qc4                                  2/2     Running            0          11m
```

### Start Example

1. PVC setup host path, like this:
```
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: csi-hostpath-pvc
  annotations:
    # setup host path
    csi-hostpath-path: /var/log
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
  storageClassName: csi-hostpath
```

2. Run example
```
# install storageclass
$ kubectl apply -f ./examples/storageclass.yaml

# install pvc
$ kubectl apply -f ./examples/pvc.yaml

# install pod
$ kubectl apply -f ./examples/nginx.yaml
```

### Undeploy

1. undeploy
```
$ helm uninstall csi-hostpath
```

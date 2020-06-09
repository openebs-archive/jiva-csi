# jiva-csi

[![Releases](https://img.shields.io/github/release/openebs/openebs/all.svg?style=flat-square)](https://github.com/openebs/openebs/releases)
[![Slack channel #openebs](https://img.shields.io/badge/slack-openebs-brightgreen.svg?logo=slack)](https://kubernetes.slack.com/messages/openebs)
[![Twitter](https://img.shields.io/twitter/follow/openebs.svg?style=social&label=Follow)](https://twitter.com/intent/follow?screen_name=openebs)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](https://github.com/openebs/openebs/blob/master/CONTRIBUTING.md)
[![Go Report Card](https://goreportcard.com/badge/github.com/openebs/jiva-csi)](https://goreportcard.com/report/github.com/openebs/jiva-csi)
[![Build Status](https://travis-ci.org/openebs/jiva-csi.svg?branch=master)](https://travis-ci.org/openebs/jiva-csi)

## Overview

Jiva CSI driver implements the [csi-spec](https://github.com/container-storage-interface/spec/blob/master/spec.md) for
the provision and deprovision of the OpenEBS Jiva volumes on kubernetes.

Jiva CSI driver comprises of 2 components:
- A controller component launched as a StatefulSet,
  implementing the CSI controller services. The Control Plane
  services are responsible for creating/deleting the required
  OpenEBS Volume.
- A node component that runs as a DaemonSet,
  implementing the CSI node services. The node component is
  responsible for performing the iSCSI connection management and
  connecting to the OpenEBS Volume.

## Quick Start

### Prerequisites

1. Kubernetes version 1.14 or higher
2. OpenEBS Version 1.5 or higher installed.
   The steps to install OpenEBS are [here](https://docs.openebs.io/docs/next/quickstart.html)
3. jiva-operator must be installed.
   The steps to install jiva-operator is [here](https://github.com/openebs/jiva-operator/blob/master/README.md)
4. iSCSI initiator utils installed on all the worker nodes
5. You have access to install RBAC components into kube-system namespace.
   The Jiva CSI driver components are installed in kube-system
   namespace to allow them to be flagged as system critical components.

### Installation

Run following commands to proceed with the installation:
- For Ubuntu 16.04.
  ```
  kubectl apply -f https://raw.githubusercontent.com/openebs/jiva-csi/master/deploy/jiva-csi-ubuntu-16.04.yaml
  ```

- For Ubuntu 18.04
  ```
  kubectl apply -f https://raw.githubusercontent.com/openebs/jiva-csi/master/deploy/jiva-csi.yaml
  ```

Verify that the Jiva CSI Components are installed.

```
$ kubectl get pods -n kube-system -l role=openebs-csi
NAME                            READY   STATUS    RESTARTS   AGE
openebs-jiva-csi-controller-0   4/4     Running   0          6m14s
openebs-jiva-csi-node-56t5g     2/2     Running   0          6m13s

```

### Provision a Jiva volume

1. Create Jiva volume policy to set various policies for creating
   jiva volume. Though this is optional as there are already some
   default values are set for some field like replicaSC, replicationFactor
   etc with default value openebs-hostpath and 3 respectively.
   A sample jiva volume policy CR looks like:
   ``` 
    apiVersion: openebs.io/v1alpha1
    kind: JivaVolumePolicy
    metadata:
      name: example-jivavolumepolicy
      namespace: openebs
    spec:
      replicaSC: openebs-hostpath
      enableBufio: false
      autoScaling: false
      target:
        # monitor: false
        replicationFactor: 1
        # auxResources:
        # tolerations:
        # resources:
        # affinity:
        # nodeSelector:
        # priorityClassName:
      # replica:
        # tolerations:
        # resources:
        # affinity:
        # nodeSelector:
        # priorityClassName:
    ```
2. Create a Storage Class to dynamically provision volumes
   using jiva-csi driver. A sample storage class looks like:
   ```
   apiVersion: storage.k8s.io/v1
   kind: StorageClass
   metadata:
     name: openebs-jiva-csi-sc
   provisioner: jiva.csi.openebs.io
   parameters:
     cas-type: "jiva"
     policy: "example-jivavolumepolicy"
   ```
2. Create PVC by specifying the above Storage Class in the PVC spec
   ```
   kind: PersistentVolumeClaim
   apiVersion: v1
   metadata:
     name: jiva-csi-demo
   spec:
     storageClassName: openebs-jiva-csi-sc
     accessModes:
       - ReadWriteOnce
     resources:
       requests:
         storage: 4Gi
   ```
4. Deploy your application by specifying the PVC name
   ```
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: fio
   spec:
     selector:
       matchLabels:
         name: fio
     replicas: 1
     strategy:
       type: Recreate
       rollingUpdate: null
     template:
       metadata:
         labels:
           name: fio
       spec:
         nodeName: gke-utkarsh-csi-default-pool-953ba289-rt9l
         containers:
         - name: perfrunner
           image: openebs/tests-fio
           command: ["/bin/bash"]
           args: ["-c", "while true ;do sleep 50; done"]
           volumeMounts:
         - mountPath: /datadir
           name: fio-vol
         volumes:
         - name: fio-vol
           persistentVolumeClaim:
             claimName: jiva-csi-demo
   ```

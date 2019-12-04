#!/bin/bash

set -ex

function waitForComponent() {
  RESOURCE=$1
  COMPONENT=$2
  NS=$3
	replicas=""

  for i in $(seq 1 50) ; do
    kubectl get $RESOURCE -n ${NS} ${COMPONENT}
		if [ "$RESOURCE" == "ds" ] || [ "$RESOURCE" == "daemonset" ];
		then
			replicas=$(kubectl get $RESOURCE -n ${NS} ${COMPONENT} -o json | jq ".status.numberReady")
		else
			replicas=$(kubectl get $RESOURCE -n ${NS} ${COMPONENT} -o json | jq ".status.readyReplicas")
		fi
    if [ "$replicas" == "1" ]; then
      break
    else
      echo "Waiting for ${COMPONENT} to be ready"
      sleep 10
      if [ $i -eq "50"];
      then
        exit 1
      fi
    fi
  done
}

waitForComponent "deploy" "openebs-ndm-operator" "openebs"
waitForComponent "ds" "openebs-ndm" "openebs"
waitForComponent "deploy" "openebs-localpv-provisioner" "openebs"
waitForComponent "sts" "openebs-jiva-csi-controller" "kube-system"
waitForComponent "ds" "openebs-jiva-csi-node" "kube-system"

SOCK_PATH=/var/lib/kubelet/pods/`kubectl get pod -n kube-system openebs-jiva-csi-controller-0 -o 'jsonpath={.metadata.uid}'`/volumes/kubernetes.io~empty-dir/socket-dir/csi.sock
chmod -R 777 /var/lib/kubelet
ln -s $SOCK_PATH /tmp/csi.sock
chmod -R 777 /tmp/csi.sock

cat <<EOT >> /tmp/parameters.json
{
        "cas-type": "jiva",
        "replicaCount": "1"
}
EOT

which csi-sanity
if [ $? != 0 ];
then
	echo "csi-sanity not found"
	exit 1
fi

csi-sanity --ginkgo.v --csi.controllerendpoint=///tmp/csi.sock --csi.endpoint=/var/lib/kubelet/plugins/jiva.csi.openebs.io/csi.sock --csi.testvolumeparameters=/tmp/parameters.json
exit 0

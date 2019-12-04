#!/bin/bash

set -ex
function waitForComponent() {
  RESOURCE=$1
  COMPONENT=$2
  NS=$3
	CONTAINER=$4
	replicas=""

  for i in $(seq 1 50) ; do
    kubectl get $RESOURCE -n ${NS} ${COMPONENT}
		if [ "$RESOURCE" == "ds" ] || [ "$RESOURCE" == "daemonset" ];
		then
			replicas=$(kubectl get $RESOURCE -n ${NS} ${COMPONENT} -o json | jq ".status.numberReady")
		else
			replicas=$(kubectl get $RESOURCE -n ${NS} ${COMPONENT} -o json | jq ".status.readyReplicas")
		fi
    if [ "$replicas" == "1" ];
		then
			echo "${COMPONENT} is ready"
      break
    else
      echo "Waiting for ${COMPONENT} to be ready"
      sleep 10
      if [ $i -eq "10" ];
      then
				kubectl describe $RESOURCE $COMPONENT -n $NS
				POD=$(kubectl get pod -n $NS -l app=openebs-jiva-csi-node -o jsonpath='{range .items[*]}{@.metadata.name}')
				PROVISIONER=$(kubectl get pod -n openebs -l name=openebs-localpv-provisioner -o jsonpath='{range .items[*]}{@.metadata.name}')
				kubectl describe pod $POD -n $NS
				if [ -n $CONTAINER ];
				then
					kubectl logs --tail=20 $PROVISIONER -n openebs
					kubectl logs --tail=20 $POD -n $NS -c $CONTAINER
					exit 1
				fi
      fi
    fi
  done
}

waitForComponent "deploy" "openebs-ndm-operator" "openebs"
waitForComponent "ds" "openebs-ndm" "openebs"
waitForComponent "deploy" "openebs-localpv-provisioner" "openebs"
waitForComponent "sts" "openebs-jiva-csi-controller" "kube-system" "openebs-jiva-csi-plugin"
waitForComponent "ds" "openebs-jiva-csi-node" "kube-system" "openebs-jiva-csi-plugin"

SOCK_PATH=/var/lib/kubelet/pods/`kubectl get pod -n kube-system openebs-jiva-csi-controller-0 -o 'jsonpath={.metadata.uid}'`/volumes/kubernetes.io~empty-dir/socket-dir/csi.sock
sudo chmod -R 777 /var/lib/kubelet
sudo ln -s $SOCK_PATH /tmp/csi.sock
sudo chmod -R 777 /tmp/csi.sock

cat <<EOT >> /tmp/parameters.json
{
        "cas-type": "jiva",
        "replicaCount": "1"
}
EOT

CSI_TEST_REPO=https://github.com/kubernetes-csi/csi-test.git
CSI_REPO_PATH="$GOPATH/src/github.com/kubernetes-csi/csi-test"

if [ ! -d "$CSI_REPO_PATH" ] ; then
       git clone $CSI_TEST_REPO $CSI_REPO_PATH
else
    cd "$CSI_REPO_PATH"
    git pull $CSI_REPO_PATH
fi

cd "$CSI_REPO_PATH/cmd/csi-sanity"
make clean
make

csi-sanity --ginkgo.v --csi.controllerendpoint=///tmp/csi.sock --csi.endpoint=/var/lib/kubelet/plugins/jiva.csi.openebs.io/csi.sock --csi.testvolumeparameters=/tmp/parameters.json
exit 0

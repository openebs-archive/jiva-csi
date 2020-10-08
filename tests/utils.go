/*
Copyright 2020 The OpenEBS Authors

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

package volume

import (
	"fmt"
	"strings"

	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func createStorageClass() {
	stdout, stderr, err := KubectlWithInput([]byte(SCYAML), "apply", "-f", "-")
	Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
}

func deleteStorageClass() {
	stdout, stderr, err := KubectlWithInput([]byte(SCYAML), "delete", "-f", "-")
	Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
}

func createJivaVolumePolicy() {
	stdout, stderr, err := KubectlWithInput([]byte(policyYAML), "apply", "-f", "-")
	Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
}

func deleteJivaVolumePolicy() {
	stdout, stderr, err := KubectlWithInput([]byte(policyYAML), "delete", "-f", "-")
	Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
}

func deletePVC() {
	stdout, stderr, err := KubectlWithInput([]byte(PVCYAML), "delete", "-n", NSName, "-f", "-")
	Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

	By("verifying pvc is deleted")
	verifyPVCDeleted(NSName, PVCName)

}

func verifyPVCDeleted(ns, pvc string) {
	var (
		err error
	)
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		_, _, err = Kubectl("get", "pvc", pvc, "-n", NSName)
		if err == nil {
			continue
		}
		break
	}
	Expect(err).NotTo(BeNil(), "not able to delete pvc")
}

func createAndVerifyPVC() {
	var (
		err error
	)
	By("creating pvc")
	stdout, stderr, err := KubectlWithInput([]byte(PVCYAML), "apply", "-n", NSName, "-f", "-")
	Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

	By("verifying pv is bound")
	verifyVolumeCreated(NSName, PVCName)
}

func verifyVolumeCreated(ns, pvc string) {
	var volName string
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		stdout, stderr, err := Kubectl("get", "pvc", "-n", ns, pvc, "-o=template", "--template={{.spec.volumeName}}")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		volName = strings.TrimSpace(string(stdout))
		if volName == "" {
			fmt.Println("Waiting for PVC to have spec.VolumeName")
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}
	Expect(volName).NotTo(BeEmpty(), "not able to get pv name from PVC.Spec.VolumeName")
}

func createDeployVerifyApp() {
	By("creating and deploying app pod", createAndDeployAppPod)
	time.Sleep(30 * time.Second)
	By("verifying app pod is running", verifyAppPodRunning)
}

func createAndDeployAppPod() {
	var err error
	By("building a busybox app pod deployment using above csi jiva volume")
	stdout, stderr, err := KubectlWithInput([]byte(DeployYAML), "apply", "-n", NSName, "-f", "-")
	Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
}

func deleteAppDeployment() {
	stdout, stderr, err := KubectlWithInput([]byte(DeployYAML), "delete", "-n", NSName, "-f", "-")
	Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
	By("verifying deployment is deleted")
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		_, _, err := Kubectl("get", "deploy", DeploymentName, "-n", NSName)
		if err == nil {
			continue
		}
		break
	}
	Expect(err).To(BeNil(), "not able to delete deployment")
}

func verifyAppPodRunning() {
	var (
		state string
	)
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		stdout, stderr, err := Kubectl("get", "po", "--selector=name=ubuntu", "-n", NSName, "-o", "jsonpath={.items[*].status.phase}")
		Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		state = strings.TrimSpace(string(stdout))
		if state != "Running" {
			fmt.Println("Waiting for app pod to be in Running state")
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}

	Expect(state).To(Equal("Running"), "while checking status of pod {%s}", "ubuntu")
}

func restartAppPodAndVerifyRunningStatus() {
	By("deleting app pod")
	stdout, stderr, err := Kubectl("delete", "po", "--selector=name=ubuntu", "-n", NSName)
	Expect(err).ShouldNot(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
	By("verifying app pod has restarted")
	verifyAppPodRunning()

}

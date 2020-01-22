/*
Copyright © 2018-2019 The OpenEBS Authors

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

package jivavolume

import (
	"errors"

	"github.com/container-storage-interface/spec/lib/go/csi"
	jv "github.com/openebs/jiva-operator/pkg/apis/openebs/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	defaultReplicaSC = "openebs-hostpath"
)

var (
	zero              int64
	defaultTargetSpec = jv.TargetSpec{
		ReplicationFactor: 3,
		AuxResources:      &corev1.ResourceRequirements{},
		PodTemplateResources: jv.PodTemplateResources{
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("0"),
					corev1.ResourceMemory: resource.MustParse("0"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("0"),
					corev1.ResourceMemory: resource.MustParse("0"),
				},
			},
			Tolerations:       []corev1.Toleration{},
			NodeSelector:      nil,
			PriorityClassName: "",
			Affinity:          &corev1.Affinity{},
		},
	}
	defaultReplicaSpec = jv.ReplicaSpec{
		PodTemplateResources: jv.PodTemplateResources{
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("0"),
					corev1.ResourceMemory: resource.MustParse("0"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("0"),
					corev1.ResourceMemory: resource.MustParse("0"),
				},
			},
			Tolerations:       nil,
			NodeSelector:      nil,
			PriorityClassName: "",
			Affinity:          &corev1.Affinity{},
		},
	}
)

// Jiva wraps the JivaVolume structure
type Jiva struct {
	jvObj *jv.JivaVolume
	Errs  []error
}

// New returns new instance of Jiva which is wrapper over JivaVolume
func New() *Jiva {
	return &Jiva{
		jvObj: &jv.JivaVolume{},
	}
}

// Instance returns the instance of JivaVolume
func (j *Jiva) Instance() *jv.JivaVolume {
	return j.jvObj
}

// Namespace returns the namespace of JivaVolume
func (j *Jiva) Namespace() string {
	return j.jvObj.Namespace
}

// WithKindAndAPIVersion defines the kind and apiversion field of JivaVolume
func (j *Jiva) WithKindAndAPIVersion(kind, apiv string) *Jiva {
	if kind != "" && apiv != "" {
		j.jvObj.Kind = kind
		j.jvObj.APIVersion = apiv
	} else {
		j.Errs = append(j.Errs,
			errors.New("failed to initialize JivaVolume: kind/apiversion or both are missing"),
		)
	}
	return j
}

// WithNameAndNamespace defines the name and ns of JivaVolume
func (j *Jiva) WithNameAndNamespace(name, ns string) *Jiva {
	if name != "" {
		j.jvObj.Name = name
		if ns != "" {
			j.jvObj.Namespace = ns
		} else {
			j.jvObj.Namespace = "openebs"
		}
	} else {
		j.Errs = append(j.Errs,
			errors.New("failed to initialize JivaVolume: name is missing"),
		)
	}
	return j
}

// WithLabels is used to set the labels in JivaVolume CR
func (j *Jiva) WithLabels(labels map[string]string) *Jiva {
	if labels != nil {
		j.jvObj.Labels = labels
	} else {
		j.Errs = append(j.Errs,
			errors.New("failed to initialize JivaVolume: labels are missing"))
	}
	return j
}

// ResourceParameters is a function type which return resource values
type ResourceParameters func(param string) string

// HasResourceParameters verifies whether resource parameters like CPU, Memory
// have been provided or not in req, if not, it returns default value (0)
func HasResourceParameters(req *csi.CreateVolumeRequest) ResourceParameters {
	return func(param string) string {
		val, ok := req.GetParameters()[param]
		if !ok {
			return "0"
		}
		return val
	}
}

// WithSpec defines the Spec field of JivaVolume
func (j *Jiva) WithSpec(spec jv.JivaVolumeSpec) *Jiva {
	j.jvObj.Spec = spec
	return j
}

// WithPV defines the PV field of JivaVolumeSpec
func (j *Jiva) WithPV(pvName string) *Jiva {
	j.jvObj.Spec.PV = pvName
	return j
}

// WithCapacity defines the Capacity field of JivaVolumeSpec
func (j *Jiva) WithCapacity(capacity string) *Jiva {
	j.jvObj.Spec.Capacity = capacity
	return j
}

// WithReplicaSC defines the ReplicaSC field of JivaVolumePolicySpec
func (j *Jiva) WithReplicaSC(scName string) *Jiva {
	if scName == "" {
		scName = defaultReplicaSC
	}
	j.jvObj.Spec.Policy.ReplicaSC = scName
	return j
}

// WithEnableBufio defines the ReplicaSC field of JivaVolumePolicySpec
func (j *Jiva) WithEnableBufio(enable bool) *Jiva {
	j.jvObj.Spec.Policy.EnableBufio = enable
	return j
}

// WithAutoScaling defines the ReplicaSC field of JivaVolumePolicySpec
func (j *Jiva) WithAutoScaling(enable bool) *Jiva {
	j.jvObj.Spec.Policy.AutoScaling = enable
	return j
}

// WithTarget defines the ReplicaSC field of JivaVolumePolicySpec
func (j *Jiva) WithTarget(target jv.TargetSpec) *Jiva {
	if target.ReplicationFactor == 0 {
		target.ReplicationFactor = defaultTargetSpec.ReplicationFactor
	}
	if target.AuxResources == nil {
		target.AuxResources = defaultTargetSpec.AuxResources
	}
	if target.Resources == nil {
		target.Resources = defaultTargetSpec.Resources
	}
	if target.Tolerations == nil {
		target.Tolerations = defaultTargetSpec.Tolerations
	}
	if target.Affinity == nil {
		target.Affinity = defaultTargetSpec.Affinity
	}
	if target.NodeSelector == nil {
		target.NodeSelector = defaultTargetSpec.NodeSelector
	}
	if target.PriorityClassName == "" {
		target.PriorityClassName = defaultTargetSpec.PriorityClassName
	}
	j.jvObj.Spec.Policy.Target = target
	return j
}

// WithReplica defines the ReplicaSC field of JivaVolumePolicySpec
func (j *Jiva) WithReplica(replica jv.ReplicaSpec) *Jiva {
	if replica.Resources == nil {
		replica.Resources = defaultReplicaSpec.Resources
	}
	if replica.Tolerations == nil {
		replica.Tolerations = defaultReplicaSpec.Tolerations
	}
	if replica.Affinity == nil {
		replica.Affinity = defaultReplicaSpec.Affinity
	}
	if replica.NodeSelector == nil {
		replica.NodeSelector = defaultReplicaSpec.NodeSelector
	}
	if replica.PriorityClassName == "" {
		replica.PriorityClassName = defaultReplicaSpec.PriorityClassName
	}
	j.jvObj.Spec.Policy.Replica = replica
	return j
}

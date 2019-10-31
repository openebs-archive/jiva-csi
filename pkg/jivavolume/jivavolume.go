package jivavolume

import (
	"errors"

	"github.com/container-storage-interface/spec/lib/go/csi"
	jv "github.com/openebs/jiva-operator/pkg/apis/openebs/v1alpha1"
)

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

// ResourceParameters is a function type which return resource values
type ResourceParameters func(param string) string

// HasResourceParameters verifies whether resource parameters like CPU, Memory
// have been provided or not in req, if not, it returns default value (0)
func HasResourceParameters(req *csi.CreateVolumeRequest) ResourceParameters {
	return func(param string) string {
		if val, ok := req.GetParameters()[param]; !ok {
			return "0"
		} else {
			return val
		}
	}
}

// WithReplicaStorageClass returns storage class
func WithReplicaStorageClass(sc string) string {
	if sc == "" {
		return "openebs-hostpath"
	}
	return sc
}

// WithSpec defines the Spec field of JivaVolume
func (j *Jiva) WithSpec(spec jv.JivaVolumeSpec) *Jiva {
	j.jvObj.Spec = spec
	return j
}

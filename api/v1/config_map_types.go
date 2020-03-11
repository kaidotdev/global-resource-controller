package v1

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigMapTemplate struct {
	// Data contains the configuration data.
	// Each key must consist of alphanumeric characters, '-', '_' or '.'.
	// Values with non-UTF-8 byte sequences must use the BinaryData field.
	// The keys stored in Data must not overlap with the keys in
	// the BinaryData field, this is enforced during validation process.
	// +optional
	Data map[string]string `json:"data,omitempty" protobuf:"bytes,2,rep,name=data"`

	// BinaryData contains the binary data.
	// Each key must consist of alphanumeric characters, '-', '_' or '.'.
	// BinaryData can contain byte sequences that are not in the UTF-8 range.
	// The keys stored in BinaryData must not overlap with the ones in
	// the Data field, this is enforced during validation process.
	// Using this field will require 1.10+ apiserver and
	// kubelet.
	// +optional
	BinaryData map[string][]byte `json:"binaryData,omitempty" protobuf:"bytes,3,rep,name=binaryData"`
}

// GlobalConfigMapSpec defines the desired state of GlobalConfigMap
type GlobalConfigMapSpec struct {
	ExcludeNamespaces []string          `json:"excludeNamespaces,omitempty"`
	Template          ConfigMapTemplate `json:"template"`
}

// GlobalConfigMapStatus defines the observed state of GlobalConfigMap
type GlobalConfigMapStatus struct {
	TargetNamespaces []string `json:"targetNamespaces,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// GlobalConfigMap is the schema for the runners API
type GlobalConfigMap struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GlobalConfigMapSpec   `json:"spec"`
	Status GlobalConfigMapStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GlobalConfigMapList contains a list of Attack
type GlobalConfigMapList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`
	Items           []GlobalConfigMap `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GlobalConfigMap{}, &GlobalConfigMapList{})
}

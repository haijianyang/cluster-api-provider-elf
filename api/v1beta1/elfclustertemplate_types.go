/*
Copyright 2022.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ElfClusterTemplateResource struct {
	Spec ElfClusterSpec `json:"spec"`
}

// ElfClusterTemplateSpec defines the desired state of ElfClusterTemplate
type ElfClusterTemplateSpec struct {
	Template ElfClusterTemplateResource `json:"template"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=elfclustertemplates,scope=Namespaced,categories=cluster-api
// +kubebuilder:storageversion

// ElfClusterTemplate is the Schema for the elfclustertemplates API
type ElfClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ElfClusterTemplateSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// ElfClusterTemplateList contains a list of ElfClusterTemplate
type ElfClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElfClusterTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ElfClusterTemplate{}, &ElfClusterTemplateList{})
}

/*
Copyright 2023.

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

package v1alpha1

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// HostConfigFinalizer is set on PrepareForCreate callback.
	HostConfigFinalizer = "hostconfig.kubesmart.smtx.io"

	// HostConfigReRunAnnotation 表示重新执行.
	HostConfigReRunAnnotation = "hostconfig.kubesmart.smtx.io/re-run"

	// HostConfigConfigHashAnnotation 记录 spec.config 哈希值的 annotation，当实际计算的哈希值与记录的不同时，代表配置有变更，需要重新执行.
	HostConfigConfigHashAnnotation = "hostconfig.kubesmart.smtx.io/config-hash"

	// HostConfigNodeNameLabel 表示属于哪个节点，如果长度长度超过 63，会用哈希值代替.
	HostConfigNodeNameLabel = "hostconfig.kubesmart.smtx.io/node-name"
)

type HostConfigSpec struct {
	NodeName string `json:"nodeName"`
	Config   Config `json:"config"`
}

type Config struct {
	// Ansible 通过 ansible playbook 完成配置
	Ansible *Ansible `json:"ansible,omitempty"`
	// Timeout 执行一次配置的超时时间
	Timeout metav1.Duration `json:"timeout,omitempty"`
}

type HostConfigStatus struct {
	// Phase 当前状态
	Phase          Phase  `json:"phase"`
	FailureReason  string `json:"failureReason,omitempty"`
	FailureMessage string `json:"failureMessage,omitempty"`
	// LastExecutionTime 最后执行的时间戳
	LastExecutionTime *metav1.Time `json:"lastExecutionTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=hostconfigs,scope=Namespaced,categories=kubesmart,shortName=hc
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="the current phase of HostConfig"
// +kubebuilder:printcolumn:name="LastExecutionTime",type="string",JSONPath=".status.lastExecutionTime",description="the last execution time"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of HostConfig"

// HostConfig is the Schema for the HostConfig API.
type HostConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HostConfigSpec   `json:"spec,omitempty"`
	Status HostConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HostConfigList contains a list of HostConfig.
type HostConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HostConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HostConfig{}, &HostConfigList{})
}

// IsConfigStable returns true if the hash of current config is equal to the hash in annotation.
func (in *HostConfig) IsConfigStable() bool {
	annotationHash := GetAnnotation(in, HostConfigConfigHashAnnotation)
	currentHash := CalculateHash(in.Spec.Config)
	return annotationHash == currentHash
}

func GetAnnotation(obj metav1.Object, key string) string {
	annotations := obj.GetAnnotations()
	value, _ := annotations[key] //nolint:gosimple
	return value
}

func CalculateHash(data interface{}) string {
	b, _ := json.Marshal(data) //nolint:errchkjson
	hashDate := sha256.Sum256(b)
	return hex.EncodeToString(hashDate[:])
}

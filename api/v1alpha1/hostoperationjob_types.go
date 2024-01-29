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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// HostOperationJobFinalizer is set on PrepareForCreate callback.
	HostOperationJobFinalizer = "hostoperationjob.kubesmart.smtx.io"

	// HostOperationJobReRunAnnotation 表示重新执行.
	HostOperationJobReRunAnnotation = "hostoperationjob.kubesmart.smtx.io/re-run"

	// HostOperationJobNodeNameLabel 表示属于哪个节点，如果长度长度超过 63，会用哈希值代替。
	HostOperationJobNodeNameLabel = "hostoperationjob.kubesmart.smtx.io/node-name"
)

type HostOperationJobSpec struct {
	NodeName  string    `json:"nodeName"`
	Operation Operation `json:"operation"`
}

type Operation struct {
	// Ansible 通过 ansible playbook 完成操作
	Ansible *Ansible `json:"ansible,omitempty"`
	// Timeout 执行一次操作的超时时间
	Timeout metav1.Duration `json:"timeout,omitempty"`
}

type HostOperationJobStatus struct {
	// Phase 当前阶段
	Phase          Phase  `json:"phase"`
	FailureReason  string `json:"failureReason,omitempty"`
	FailureMessage string `json:"failureMessage,omitempty"`
	// LastExecutionTime 最后执行的时间戳
	LastExecutionTime *metav1.Time `json:"lastExecutionTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=hostoperationjobs,scope=Namespaced,categories=kubesmart,shortName=hoj
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="the current phase of HostOperationJob"
// +kubebuilder:printcolumn:name="LastExecutionTime",type="string",JSONPath=".status.lastExecutionTime",description="the last execution time"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of HostOperationJob"

// HostOperationJob is the Schema for the HostOperationJob API.
type HostOperationJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HostOperationJobSpec   `json:"spec,omitempty"`
	Status HostOperationJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HostOperationJobList contains a list of HostOperationJob.
type HostOperationJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HostOperationJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HostOperationJob{}, &HostOperationJobList{})
}

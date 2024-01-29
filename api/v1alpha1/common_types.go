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
	corev1 "k8s.io/api/core/v1"
)

type Phase string

// xx.
const (
	PhaseInitializing = Phase("Initializing")
	PhaseProcessing   = Phase("Processing")
	PhaseSucceeded    = Phase("Succeeded")
	PhaseFailed       = Phase("Failed")
)

type Ansible struct {
	// RemotePlaybook 在远端的 playbook，单个 .tar.gz 压缩包，内容可以是单个 yaml 文件，也可以符合 ansible 要求的目录
	RemotePlaybook *RemotePlaybook `json:"remotePlaybook,omitempty"`
	// LocalPlaybook 本地的 playbook，单个 yaml 文件， secret 引用或者 yaml 字符串
	LocalPlaybook *YAMLText `json:"localPlaybook,omitempty"`
	// Values 执行 playbook 的参数，yaml 格式，可以是 secret 引用或者 yaml 字符串
	Values *YAMLText `json:"values,omitempty"`
}

type RemotePlaybook struct {
	// URL playbook 在远端的地址，支持 https
	URL string `json:"url"`
	// Name 要执行的 playbook 文件名，相对于压缩包顶层的位置
	Name string `json:"name"`
	// MD5sum 压缩包的 MD5，填写了会进行校验，已经下载过的 playbook 校验通过后跳过重复下载
	MD5sum string `json:"md5sum,omitempty"`
}

type YAMLText struct {
	// SecretRef specifies the secret which stores yaml text.
	SecretRef *corev1.SecretReference `json:"secretRef,omitempty"`
	// Content is the inline yaml text.
	//+kubebuilder:validation:Format=yaml
	Content string `json:"content,omitempty"`
}

/*
Copyright 2024.
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

package hostagent

import (
	goctx "context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	capiutil "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentv1 "github.com/smartxworks/cluster-api-provider-elf/api/v1alpha1"
	infrav1 "github.com/smartxworks/cluster-api-provider-elf/api/v1beta1"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/hostagent/tasks"
)

const defaultTimeout = 1 * time.Minute

func GetHostJob(ctx goctx.Context, c client.Client, namespace, name string) (*agentv1.HostOperationJob, error) {
	var restartKubeletJob agentv1.HostOperationJob
	if err := c.Get(ctx, apitypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, &restartKubeletJob); err != nil {
		return nil, err
	}

	return &restartKubeletJob, nil
}

func GetAddNewDiskCapacityToRootName(elfMachine *infrav1.ElfMachine) string {
	return fmt.Sprintf("cape-expand-root-rartition-%s-%d", elfMachine.Name, elfMachine.Spec.DiskGiB)
}

func AddNewDiskCapacityToRoot(ctx goctx.Context, c client.Client, elfMachine *infrav1.ElfMachine) (*agentv1.HostOperationJob, error) {
	agentJob := &agentv1.HostOperationJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("cape-expand-root-rartition-%s-%d", elfMachine.Name, elfMachine.Spec.DiskGiB),
			Namespace: elfMachine.Namespace,
		},
		Spec: agentv1.HostOperationJobSpec{
			NodeName: elfMachine.Name,
			Operation: agentv1.Operation{
				Ansible: &agentv1.Ansible{
					LocalPlaybook: &agentv1.YAMLText{
						Content: tasks.ExpandRootPartitionTask,
					},
				},
				Timeout: metav1.Duration{Duration: defaultTimeout},
			},
		},
	}

	if err := c.Create(ctx, agentJob); err != nil {
		return nil, err
	}

	return agentJob, nil
}

func RestartMachineKubelet(ctx goctx.Context, c client.Client, elfMachine *infrav1.ElfMachine) (*agentv1.HostOperationJob, error) {
	restartKubeletJob := &agentv1.HostOperationJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("cape-restart-kubelet-%s-%s", elfMachine.Name, capiutil.RandomString(6)),
			Namespace: elfMachine.Namespace,
		},
		Spec: agentv1.HostOperationJobSpec{
			NodeName: elfMachine.Name,
			Operation: agentv1.Operation{
				Ansible: &agentv1.Ansible{
					LocalPlaybook: &agentv1.YAMLText{
						Content: tasks.RestartKubeletTask,
					},
				},
				Timeout: metav1.Duration{Duration: defaultTimeout},
			},
		},
	}

	if err := c.Create(ctx, restartKubeletJob); err != nil {
		return nil, err
	}

	return restartKubeletJob, nil
}

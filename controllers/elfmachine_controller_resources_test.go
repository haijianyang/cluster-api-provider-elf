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

package controllers

import (
	"bytes"
	goctx "context"
	"time"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	agentv1 "github.com/smartxworks/host-config-agent-api/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/smartxworks/cluster-api-provider-elf/api/v1beta1"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/hostagent"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/hostagent/tasks"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/service"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/service/mock_services"
	"github.com/smartxworks/cluster-api-provider-elf/test/fake"
	"github.com/smartxworks/cluster-api-provider-elf/test/helpers"
)

var _ = Describe("ElfMachineReconciler", func() {
	var (
		elfCluster       *infrav1.ElfCluster
		cluster          *clusterv1.Cluster
		elfMachine       *infrav1.ElfMachine
		machine          *clusterv1.Machine
		secret           *corev1.Secret
		kubeConfigSecret *corev1.Secret
		logBuffer        *bytes.Buffer
		mockCtrl         *gomock.Controller
		mockVMService    *mock_services.MockVMService
		mockNewVMService service.NewVMServiceFunc
	)

	BeforeEach(func() {
		logBuffer = new(bytes.Buffer)
		klog.SetOutput(logBuffer)

		elfCluster, cluster, elfMachine, machine, secret = fake.NewClusterAndMachineObjects()

		// mock
		mockCtrl = gomock.NewController(GinkgoT())
		mockVMService = mock_services.NewMockVMService(mockCtrl)
		mockNewVMService = func(_ goctx.Context, _ infrav1.Tower, _ logr.Logger) (service.VMService, error) {
			return mockVMService, nil
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("expandVMRootPartition", func() {
		BeforeEach(func() {
			var err error
			kubeConfigSecret, err = helpers.NewKubeConfigSecret(testEnv, cluster.Namespace, cluster.Name)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should not expand root partition without ResourcesHotUpdatedCondition", func() {
			ctrlMgrCtx := fake.NewControllerManagerContext(elfCluster, cluster, elfMachine, machine, secret, kubeConfigSecret)
			fake.InitOwnerReferences(ctx, ctrlMgrCtx, elfCluster, cluster, elfMachine, machine)
			machineContext := newMachineContext(elfCluster, cluster, elfMachine, machine, mockVMService)

			reconciler := &ElfMachineReconciler{ControllerManagerContext: ctrlMgrCtx, NewVMService: mockNewVMService}
			ok, err := reconciler.expandVMRootPartition(ctx, machineContext)
			Expect(ok).To(BeTrue())
			Expect(err).NotTo(HaveOccurred())
			expectConditions(elfMachine, []conditionAssertion{})
		})

		It("should create agent job to expand root partition", func() {
			conditions.MarkFalse(elfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.ExpandingVMDiskReason, clusterv1.ConditionSeverityInfo, "")
			ctrlMgrCtx := fake.NewControllerManagerContext(elfCluster, cluster, elfMachine, machine, secret, kubeConfigSecret)
			fake.InitOwnerReferences(ctx, ctrlMgrCtx, elfCluster, cluster, elfMachine, machine)
			machineContext := newMachineContext(elfCluster, cluster, elfMachine, machine, mockVMService)

			reconciler := &ElfMachineReconciler{ControllerManagerContext: ctrlMgrCtx, NewVMService: mockNewVMService}
			ok, err := reconciler.expandVMRootPartition(ctx, machineContext)
			Expect(ok).To(BeFalse())
			Expect(err).NotTo(HaveOccurred())
			Expect(logBuffer.String()).To(ContainSubstring("Waiting for expanding root partition"))
			expectConditions(elfMachine, []conditionAssertion{{infrav1.ResourcesHotUpdatedCondition, corev1.ConditionFalse, clusterv1.ConditionSeverityInfo, infrav1.ExpandingRootPartitionReason}})
			agentJob, err := hostagent.GetHostJob(ctx, testEnv.Client, elfMachine.Namespace, hostagent.GetExpandRootPartitionJobName(elfMachine))
			Expect(err).NotTo(HaveOccurred())
			Expect(agentJob.Name).To(Equal(hostagent.GetExpandRootPartitionJobName(elfMachine)))
		})

		It("should retry when job failed", func() {
			conditions.MarkFalse(elfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.ExpandingVMDiskReason, clusterv1.ConditionSeverityInfo, "")
			agentJob := newExpandRootPartitionJob(elfMachine)
			Expect(testEnv.CreateAndWait(ctx, agentJob)).NotTo(HaveOccurred())
			ctrlMgrCtx := fake.NewControllerManagerContext(elfCluster, cluster, elfMachine, machine, secret, kubeConfigSecret)
			fake.InitOwnerReferences(ctx, ctrlMgrCtx, elfCluster, cluster, elfMachine, machine)
			machineContext := newMachineContext(elfCluster, cluster, elfMachine, machine, mockVMService)
			reconciler := &ElfMachineReconciler{ControllerManagerContext: ctrlMgrCtx, NewVMService: mockNewVMService}
			ok, err := reconciler.expandVMRootPartition(ctx, machineContext)
			Expect(ok).To(BeFalse())
			Expect(err).NotTo(HaveOccurred())
			Expect(logBuffer.String()).To(ContainSubstring("Waiting for expanding root partition job done"))
			expectConditions(elfMachine, []conditionAssertion{{infrav1.ResourcesHotUpdatedCondition, corev1.ConditionFalse, clusterv1.ConditionSeverityInfo, infrav1.ExpandingRootPartitionReason}})

			logBuffer.Reset()
			Expect(testEnv.Get(ctx, client.ObjectKey{Namespace: agentJob.Namespace, Name: agentJob.Name}, agentJob)).NotTo(HaveOccurred())
			agentJobPatchSource := agentJob.DeepCopy()
			agentJob.Status.Phase = agentv1.PhaseFailed
			Expect(testEnv.PatchAndWait(ctx, agentJob, agentJobPatchSource)).To(Succeed())
			ok, err = reconciler.expandVMRootPartition(ctx, machineContext)
			Expect(ok).To(BeFalse())
			Expect(err).NotTo(HaveOccurred())
			Expect(logBuffer.String()).To(ContainSubstring("Expand root partition failed, will try again"))
			expectConditions(elfMachine, []conditionAssertion{{infrav1.ResourcesHotUpdatedCondition, corev1.ConditionFalse, clusterv1.ConditionSeverityWarning, infrav1.ExpandingRootPartitionFailedReason}})

			Expect(testEnv.Get(ctx, client.ObjectKey{Namespace: agentJob.Namespace, Name: agentJob.Name}, agentJob)).NotTo(HaveOccurred())
			agentJobPatchSource = agentJob.DeepCopy()
			agentJob.Status.LastExecutionTime = &metav1.Time{Time: time.Now().Add(-3 * time.Minute).UTC()}
			Expect(testEnv.PatchAndWait(ctx, agentJob, agentJobPatchSource)).To(Succeed())
			ok, err = reconciler.expandVMRootPartition(ctx, machineContext)
			Expect(ok).To(BeFalse())
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				err := testEnv.Get(ctx, client.ObjectKey{Namespace: agentJob.Namespace, Name: agentJob.Name}, agentJob)
				return apierrors.IsNotFound(err)
			}, 10*time.Second).Should(BeTrue())
		})

		It("should retry when job failed", func() {
			conditions.MarkFalse(elfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.ExpandingVMDiskReason, clusterv1.ConditionSeverityInfo, "")
			agentJob := newExpandRootPartitionJob(elfMachine)
			Expect(testEnv.CreateAndWait(ctx, agentJob)).NotTo(HaveOccurred())
			Expect(testEnv.Get(ctx, client.ObjectKey{Namespace: agentJob.Namespace, Name: agentJob.Name}, agentJob)).NotTo(HaveOccurred())
			agentJobPatchSource := agentJob.DeepCopy()
			agentJob.Status.Phase = agentv1.PhaseSucceeded
			Expect(testEnv.PatchAndWait(ctx, agentJob, agentJobPatchSource)).To(Succeed())
			ctrlMgrCtx := fake.NewControllerManagerContext(elfCluster, cluster, elfMachine, machine, secret, kubeConfigSecret)
			fake.InitOwnerReferences(ctx, ctrlMgrCtx, elfCluster, cluster, elfMachine, machine)
			machineContext := newMachineContext(elfCluster, cluster, elfMachine, machine, mockVMService)
			reconciler := &ElfMachineReconciler{ControllerManagerContext: ctrlMgrCtx, NewVMService: mockNewVMService}
			ok, err := reconciler.expandVMRootPartition(ctx, machineContext)
			Expect(ok).To(BeTrue())
			Expect(err).NotTo(HaveOccurred())
			Expect(logBuffer.String()).To(ContainSubstring("Expand root partition to root succeeded"))
			expectConditions(elfMachine, []conditionAssertion{{infrav1.ResourcesHotUpdatedCondition, corev1.ConditionFalse, clusterv1.ConditionSeverityInfo, infrav1.ExpandingRootPartitionReason}})
		})
	})
})

func newExpandRootPartitionJob(elfMachine *infrav1.ElfMachine) *agentv1.HostOperationJob {
	return &agentv1.HostOperationJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hostagent.GetExpandRootPartitionJobName(elfMachine),
			Namespace: "default",
		},
		Spec: agentv1.HostOperationJobSpec{
			NodeName: elfMachine.Name,
			Operation: agentv1.Operation{
				Ansible: &agentv1.Ansible{
					LocalPlaybookText: &agentv1.YAMLText{
						Inline: tasks.ExpandRootPartitionTask,
					},
				},
			},
		},
	}
}

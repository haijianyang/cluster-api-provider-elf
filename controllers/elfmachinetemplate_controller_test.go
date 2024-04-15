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

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/smartxworks/cluster-api-provider-elf/api/v1beta1"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/context"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/service"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/service/mock_services"
	"github.com/smartxworks/cluster-api-provider-elf/test/fake"
)

var _ = Describe("ElfMachineTemplateReconciler", func() {
	var (
		elfCluster       *infrav1.ElfCluster
		cluster          *clusterv1.Cluster
		elfMachine       *infrav1.ElfMachine
		machine          *clusterv1.Machine
		secret           *corev1.Secret
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

	Context("Reconcile a ElfMachineTemplate", func() {
		It("", func() {
		})
	})

	Context("preflightChecksForCP", func() {
		// *kcp.Spec.Replicas > kcp.Status.UpdatedReplicas && *kcp.Spec.Replicas <= kcp.Status.Replicas
		It("return false if KCP rolling update in progress", func() {
			emt := fake.NewElfMachineTemplate()
			kcp := fake.NewKCP()
			kcp.Spec.Replicas = 
			ctrlMgrCtx := fake.NewControllerManagerContext(elfCluster, cluster, elfMachine, machine, secret)
			fake.InitOwnerReferences(ctx, ctrlMgrCtx, elfCluster, cluster, elfMachine, machine)
			mtCtx := &context.MachineTemplateContext{
				Cluster:            cluster,
				ElfCluster:         elfCluster,
				ElfMachineTemplate: emt,
				VMService:          mockVMService,
			}
			reconciler := &ElfMachineTemplateReconciler{ControllerManagerContext: ctrlMgrCtx, NewVMService: mockNewVMService}
			ok, err := reconciler.preflightChecksForCP(ctx, mtCtx, kcp)
			Expect(ok).NotTo(BeFalse())
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("selectResourcesNotUpToDateElfMachines", func() {
		It("should return updating/needUpdated resources elfMachines", func() {
			emt := fake.NewElfMachineTemplate()
			upToDateElfMachine, upToDateMachine := fake.NewMachineObjects(elfCluster, cluster)
			fake.SetElfMachineTemplateForElfMachine(upToDateElfMachine, emt)
			noUpToDateElfMachine, noUpToDateMachine := fake.NewMachineObjects(elfCluster, cluster)
			fake.SetElfMachineTemplateForElfMachine(noUpToDateElfMachine, emt)
			noUpToDateElfMachine.Spec.DiskGiB -= 1
			updatingElfMachine, updatingMachine := fake.NewMachineObjects(elfCluster, cluster)
			fake.SetElfMachineTemplateForElfMachine(updatingElfMachine, emt)
			conditions.MarkFalse(updatingElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.WaitingForResourcesHotUpdateReason, clusterv1.ConditionSeverityInfo, "")
			failedElfMachine, failedMachine := fake.NewMachineObjects(elfCluster, cluster)
			fake.SetElfMachineTemplateForElfMachine(failedElfMachine, emt)
			failedElfMachine.Spec.DiskGiB -= 1
			failedMachine.Status.Phase = string(clusterv1.MachinePhaseFailed)
			ctrlMgrCtx := fake.NewControllerManagerContext(elfCluster, cluster, elfMachine, machine, secret,
				upToDateElfMachine, upToDateMachine,
				noUpToDateElfMachine, noUpToDateMachine,
				updatingElfMachine, updatingMachine,
				failedElfMachine, failedMachine,
			)
			fake.InitOwnerReferences(ctx, ctrlMgrCtx, elfCluster, cluster, elfMachine, machine)
			fake.InitOwnerReferences(ctx, ctrlMgrCtx, elfCluster, cluster, upToDateElfMachine, upToDateMachine)
			fake.InitOwnerReferences(ctx, ctrlMgrCtx, elfCluster, cluster, noUpToDateElfMachine, noUpToDateMachine)
			fake.InitOwnerReferences(ctx, ctrlMgrCtx, elfCluster, cluster, updatingElfMachine, updatingMachine)
			fake.InitOwnerReferences(ctx, ctrlMgrCtx, elfCluster, cluster, failedElfMachine, failedMachine)
			reconciler := &ElfMachineTemplateReconciler{ControllerManagerContext: ctrlMgrCtx, NewVMService: mockNewVMService}
			elfMachines := []*infrav1.ElfMachine{upToDateElfMachine, noUpToDateElfMachine, updatingElfMachine, failedElfMachine}
			updatingResourcesElfMachines, needUpdatedResourcesElfMachines, err := reconciler.selectResourcesNotUpToDateElfMachines(ctx, emt, elfMachines)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatingResourcesElfMachines).To(Equal([]*infrav1.ElfMachine{updatingElfMachine}))
			Expect(needUpdatedResourcesElfMachines).To(Equal([]*infrav1.ElfMachine{noUpToDateElfMachine}))
		})
	})

	Context("markElfMachinesToBeUpdatedResources", func() {
		It("should mark resources to be updated", func() {
			emt := fake.NewElfMachineTemplate()
			fake.SetElfMachineTemplateForElfMachine(elfMachine, emt)
			elfMachine.Spec.DiskGiB -= 1
			ctrlMgrCtx := fake.NewControllerManagerContext(elfCluster, cluster, elfMachine, machine, secret)
			fake.InitOwnerReferences(ctx, ctrlMgrCtx, elfCluster, cluster, elfMachine, machine)
			reconciler := &ElfMachineTemplateReconciler{ControllerManagerContext: ctrlMgrCtx, NewVMService: mockNewVMService}
			err := reconciler.markElfMachinesToBeUpdatedResources(ctx, emt, []*infrav1.ElfMachine{elfMachine})
			Expect(err).NotTo(HaveOccurred())
			Expect(logBuffer.String()).To(ContainSubstring("Resources of ElfMachine is not up to date, marking for updating resources"))
			elfMachineKey := client.ObjectKey{Namespace: elfMachine.Namespace, Name: elfMachine.Name}
			Eventually(func() bool {
				_ = reconciler.Client.Get(ctx, elfMachineKey, elfMachine)
				return elfMachine.Spec.DiskGiB == emt.Spec.Template.Spec.DiskGiB
			}, timeout).Should(BeTrue())
			expectConditions(elfMachine, []conditionAssertion{{infrav1.ResourcesHotUpdatedCondition, corev1.ConditionFalse, clusterv1.ConditionSeverityInfo, infrav1.WaitingForResourcesHotUpdateReason, ""}})
		})
	})

	Context("markElfMachinesResourcesNotUpToDate", func() {
		It("should mark resources not up to date", func() {
			ctrlMgrCtx := fake.NewControllerManagerContext(elfCluster, cluster, elfMachine, machine, secret)
			emt := fake.NewElfMachineTemplate()
			fake.SetElfMachineTemplateForElfMachine(elfMachine, emt)
			fake.InitOwnerReferences(ctx, ctrlMgrCtx, elfCluster, cluster, elfMachine, machine)
			reconciler := &ElfMachineTemplateReconciler{ControllerManagerContext: ctrlMgrCtx, NewVMService: mockNewVMService}
			err := reconciler.markElfMachinesResourcesNotUpToDate(ctx, emt, []*infrav1.ElfMachine{elfMachine})
			Expect(err).NotTo(HaveOccurred())
			expectConditions(elfMachine, []conditionAssertion{})

			logBuffer.Reset()
			elfMachine.Spec.DiskGiB -= 1
			ctrlMgrCtx = fake.NewControllerManagerContext(elfCluster, cluster, elfMachine, machine, secret)
			fake.InitOwnerReferences(ctx, ctrlMgrCtx, elfCluster, cluster, elfMachine, machine)
			reconciler = &ElfMachineTemplateReconciler{ControllerManagerContext: ctrlMgrCtx, NewVMService: mockNewVMService}
			err = reconciler.markElfMachinesResourcesNotUpToDate(ctx, emt, []*infrav1.ElfMachine{elfMachine})
			Expect(err).NotTo(HaveOccurred())
			Expect(logBuffer.String()).To(ContainSubstring("Resources of ElfMachine is not up to date, marking for resources not up to date and waiting for hot updating resources"))
			elfMachineKey := client.ObjectKey{Namespace: elfMachine.Namespace, Name: elfMachine.Name}
			Eventually(func() bool {
				_ = reconciler.Client.Get(ctx, elfMachineKey, elfMachine)
				return elfMachine.Spec.DiskGiB == emt.Spec.Template.Spec.DiskGiB
			}, timeout).Should(BeTrue())
			expectConditions(elfMachine, []conditionAssertion{{infrav1.ResourcesHotUpdatedCondition, corev1.ConditionFalse, clusterv1.ConditionSeverityInfo, infrav1.WaitingForResourcesHotUpdateReason, anotherMachineHotUpdateInProgressMessage}})
		})
	})
})

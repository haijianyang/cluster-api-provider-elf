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

package controllers

import (
	"bytes"
	goctx "context"
	"flag"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capiutil "sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	infrav1 "github.com/smartxworks/cluster-api-provider-elf/api/v1beta1"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/context"
	towerresources "github.com/smartxworks/cluster-api-provider-elf/pkg/resources"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/service"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/service/mock_services"
	"github.com/smartxworks/cluster-api-provider-elf/test/fake"
)

const (
	Timeout = time.Second * 30
)

var _ = Describe("ElfClusterReconciler", func() {
	var (
		elfCluster       *infrav1.ElfCluster
		cluster          *clusterv1.Cluster
		logBuffer        *bytes.Buffer
		mockCtrl         *gomock.Controller
		mockVMService    *mock_services.MockVMService
		mockNewVMService service.NewVMServiceFunc
	)

	ctx := goctx.Background()

	BeforeEach(func() {
		// set log
		if err := flag.Set("logtostderr", "false"); err != nil {
			_ = fmt.Errorf("Error setting logtostderr flag")
		}
		if err := flag.Set("v", "6"); err != nil {
			_ = fmt.Errorf("Error setting v flag")
		}
		logBuffer = new(bytes.Buffer)
		klog.SetOutput(logBuffer)

		elfCluster, cluster = fake.NewClusterObjects()
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

	Context("Reconcile an ElfCluster", func() {
		It("should not reconcile when ElfCluster not found", func() {
			ctrlMgrContext := fake.NewControllerManagerContext()
			ctrlContext := &context.ControllerContext{
				ControllerManagerContext: ctrlMgrContext,
				Logger:                   ctrllog.Log,
			}

			reconciler := &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: capiutil.ObjectKey(elfCluster)})
			Expect(err).ToNot(HaveOccurred())
			Expect(result.RequeueAfter).To(BeZero())
			Expect(logBuffer.String()).To(ContainSubstring("ElfCluster not found, won't reconcile"))
		})

		It("should not error and not requeue the request without cluster", func() {
			ctrlMgrContext := fake.NewControllerManagerContext(elfCluster)
			ctrlContext := &context.ControllerContext{
				ControllerManagerContext: ctrlMgrContext,
				Logger:                   ctrllog.Log,
			}

			reconciler := &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: capiutil.ObjectKey(elfCluster)})
			Expect(err).ToNot(HaveOccurred())
			Expect(result.RequeueAfter).To(BeZero())
			Expect(logBuffer.String()).To(ContainSubstring("Waiting for Cluster Controller to set OwnerRef on ElfCluster"))
		})

		It("should not error and not requeue the request when Cluster is paused", func() {
			cluster.Spec.Paused = true

			ctrlMgrContext := fake.NewControllerManagerContext(cluster, elfCluster)
			ctrlContext := &context.ControllerContext{
				ControllerManagerContext: ctrlMgrContext,
				Logger:                   ctrlMgrContext.Logger,
			}
			fake.InitClusterOwnerReferences(ctrlContext, elfCluster, cluster)

			reconciler := &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: capiutil.ObjectKey(elfCluster)})
			Expect(err).ToNot(HaveOccurred())
			Expect(result.RequeueAfter).To(BeZero())
			Expect(logBuffer.String()).To(ContainSubstring("ElfCluster linked to a cluster that is paused"))
		})

		It("should add finalizer to the elfcluster", func() {
			elfCluster.Spec.ControlPlaneEndpoint.Host = "127.0.0.1"
			elfCluster.Spec.ControlPlaneEndpoint.Port = 6443
			ctrlMgrContext := fake.NewControllerManagerContext(cluster, elfCluster)
			ctrlContext := &context.ControllerContext{
				ControllerManagerContext: ctrlMgrContext,
				Logger:                   ctrllog.Log,
			}
			fake.InitClusterOwnerReferences(ctrlContext, elfCluster, cluster)

			mockVMService.EXPECT().FindVMsByName(cluster.Name).Return(nil, nil)

			elfClusterKey := capiutil.ObjectKey(elfCluster)
			reconciler := &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			_, _ = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: elfClusterKey})
			Expect(reconciler.Client.Get(reconciler, elfClusterKey, elfCluster)).To(Succeed())
			Expect(elfCluster.Status.Ready).To(BeTrue())
			Expect(elfCluster.Finalizers).To(ContainElement(infrav1.ClusterFinalizer))
			expectConditions(elfCluster, []conditionAssertion{
				{conditionType: clusterv1.ReadyCondition, status: corev1.ConditionTrue},
				{conditionType: infrav1.ControlPlaneEndpointReadyCondition, status: corev1.ConditionTrue},
				{conditionType: infrav1.TowerAvailableCondition, status: corev1.ConditionTrue},
			})
		})

		It("should not reconcile if without ControlPlaneEndpoint", func() {
			ctrlMgrContext := fake.NewControllerManagerContext(cluster, elfCluster)
			ctrlContext := &context.ControllerContext{
				ControllerManagerContext: ctrlMgrContext,
				Logger:                   ctrllog.Log,
			}
			fake.InitClusterOwnerReferences(ctrlContext, elfCluster, cluster)

			reconciler := &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: capiutil.ObjectKey(elfCluster)})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeZero())
			Expect(logBuffer.String()).To(ContainSubstring("The ControlPlaneEndpoint of ElfCluster is not set"))
			Expect(reconciler.Client.Get(reconciler, capiutil.ObjectKey(elfCluster), elfCluster)).To(Succeed())
			expectConditions(elfCluster, []conditionAssertion{
				{clusterv1.ReadyCondition, corev1.ConditionFalse, clusterv1.ConditionSeverityInfo, infrav1.WaitingForVIPReason},
				{infrav1.ControlPlaneEndpointReadyCondition, corev1.ConditionFalse, clusterv1.ConditionSeverityInfo, infrav1.WaitingForVIPReason},
				{conditionType: infrav1.TowerAvailableCondition, status: corev1.ConditionTrue},
			})
		})

		It("should delete duplicate virtual machines", func() {
			elfMachine1, machine1 := fake.NewMachineObjects(elfCluster, cluster)
			elfMachine2, machine2 := fake.NewMachineObjects(elfCluster, cluster)
			vm11 := fake.NewTowerVMFromElfMachine(elfMachine1)
			vm11.Status = models.NewVMStatus(models.VMStatusSTOPPED)
			vm11.EntityAsyncStatus = nil
			elfMachine1.Status.VMRef = *vm11.LocalID
			vm12 := fake.NewTowerVMFromElfMachine(elfMachine1)
			vm12.Status = models.NewVMStatus(models.VMStatusSTOPPED)
			vm12.EntityAsyncStatus = nil
			vm21 := fake.NewTowerVMFromElfMachine(elfMachine2)
			vm22 := fake.NewTowerVMFromElfMachine(elfMachine2)
			vm31 := fake.NewTowerVM()
			vm32 := fake.NewTowerVM()
			*vm32.Name = *vm31.Name
			elfCluster.Spec.ControlPlaneEndpoint.Host = "127.0.0.1"
			elfCluster.Spec.ControlPlaneEndpoint.Port = 6443
			ctrlMgrContext := fake.NewControllerManagerContext(elfCluster, cluster, elfMachine1, machine1, elfMachine2, machine2)
			ctrlContext := &context.ControllerContext{
				ControllerManagerContext: ctrlMgrContext,
				Logger:                   ctrllog.Log,
			}
			fake.InitOwnerReferences(ctrlContext, elfCluster, cluster, elfMachine1, machine1)
			fake.InitOwnerReferences(ctrlContext, elfCluster, cluster, elfMachine2, machine2)
			fake.InitClusterOwnerReferences(ctrlContext, elfCluster, cluster)

			mockVMService.EXPECT().FindVMsByName(cluster.Name).Return([]*models.VM{vm11, vm12, vm21, vm22, vm31, vm32}, nil)
			mockVMService.EXPECT().Delete(*vm12.ID).Return(nil, errors.New("some error"))

			elfClusterKey := capiutil.ObjectKey(elfCluster)
			reconciler := &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: elfClusterKey})
			Expect(result).To(BeZero())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("some error"))
			Expect(logBuffer.String()).To(ContainSubstring("Waiting for ElfMachine to select one of the duplicate VMs before deleting the other"))

			elfMachine1.Status.VMRef = *vm12.LocalID
			ctrlMgrContext = fake.NewControllerManagerContext(elfCluster, cluster, elfMachine1, machine1, elfMachine2, machine2)
			ctrlContext = &context.ControllerContext{
				ControllerManagerContext: ctrlMgrContext,
				Logger:                   ctrllog.Log,
			}
			fake.InitOwnerReferences(ctrlContext, elfCluster, cluster, elfMachine1, machine1)
			fake.InitOwnerReferences(ctrlContext, elfCluster, cluster, elfMachine2, machine2)
			fake.InitClusterOwnerReferences(ctrlContext, elfCluster, cluster)
			mockVMService.EXPECT().FindVMsByName(cluster.Name).Return([]*models.VM{vm11, vm12}, nil)
			mockVMService.EXPECT().Delete(*vm11.ID).Return(nil, errors.New("some error"))
			elfClusterKey = capiutil.ObjectKey(elfCluster)
			reconciler = &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			result, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: elfClusterKey})
			Expect(result).To(BeZero())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("some error"))
		})
	})

	Context("Delete a ElfCluster", func() {
		BeforeEach(func() {
			ctrlutil.AddFinalizer(elfCluster, infrav1.ClusterFinalizer)
			elfCluster.DeletionTimestamp = &metav1.Time{Time: time.Now().UTC()}
		})

		It("should not remove elfcluster finalizer when has elfmachines", func() {
			elfMachine, machine := fake.NewMachineObjects(elfCluster, cluster)
			ctrlMgrContext := fake.NewControllerManagerContext(elfCluster, cluster, elfMachine, machine)
			ctrlContext := &context.ControllerContext{
				ControllerManagerContext: ctrlMgrContext,
				Logger:                   ctrllog.Log,
			}
			fake.InitOwnerReferences(ctrlContext, elfCluster, cluster, elfMachine, machine)

			reconciler := &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			elfClusterKey := capiutil.ObjectKey(elfCluster)
			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: elfClusterKey})
			Expect(logBuffer.String()).To(ContainSubstring("Waiting for ElfMachines to be deleted"))
			Expect(result.RequeueAfter).NotTo(BeZero())
			Expect(err).ShouldNot(HaveOccurred())
			Expect(reconciler.Client.Get(reconciler, elfClusterKey, elfCluster)).To(Succeed())
			Expect(elfCluster.Finalizers).To(ContainElement(infrav1.ClusterFinalizer))
		})

		It("should delete labels and remove elfcluster finalizer", func() {
			task := fake.NewTowerTask()
			ctrlMgrContext := fake.NewControllerManagerContext(cluster, elfCluster)
			ctrlContext := &context.ControllerContext{
				ControllerManagerContext: ctrlMgrContext,
				Logger:                   ctrllog.Log,
			}
			fake.InitClusterOwnerReferences(ctrlContext, elfCluster, cluster)

			reconciler := &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			elfClusterKey := capiutil.ObjectKey(elfCluster)

			mockVMService.EXPECT().DeleteVMPlacementGroupsByName(towerresources.GetVMPlacementGroupNamePrefix(cluster)).Return(errors.New("some error"))
			mockVMService.EXPECT().FindVMsByName(cluster.Name).Return(nil, nil)

			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: elfClusterKey})
			Expect(result).To(BeZero())
			Expect(err).To(HaveOccurred())

			logBuffer = new(bytes.Buffer)
			klog.SetOutput(logBuffer)
			task.Status = models.NewTaskStatus(models.TaskStatusSUCCESSED)
			mockVMService.EXPECT().DeleteVMPlacementGroupsByName(towerresources.GetVMPlacementGroupNamePrefix(cluster)).Return(nil)
			mockVMService.EXPECT().DeleteLabel(towerresources.GetVMLabelClusterName(), elfCluster.Name, true).Return("labelid", nil)
			mockVMService.EXPECT().DeleteLabel(towerresources.GetVMLabelVIP(), elfCluster.Spec.ControlPlaneEndpoint.Host, false).Return("labelid", nil)
			mockVMService.EXPECT().DeleteLabel(towerresources.GetVMLabelNamespace(), elfCluster.Namespace, true).Return("", nil)
			mockVMService.EXPECT().DeleteLabel(towerresources.GetVMLabelManaged(), "true", true).Return("", nil)

			result, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: elfClusterKey})
			Expect(result).To(BeZero())
			Expect(err).NotTo(HaveOccurred())
			Expect(logBuffer.String()).To(ContainSubstring(fmt.Sprintf("Placement groups with name prefix %s deleted", towerresources.GetVMPlacementGroupNamePrefix(cluster))))
			Expect(logBuffer.String()).To(ContainSubstring(fmt.Sprintf("Label %s:%s deleted", towerresources.GetVMLabelClusterName(), elfCluster.Name)))
			Expect(apierrors.IsNotFound(reconciler.Client.Get(reconciler, elfClusterKey, elfCluster))).To(BeTrue())
		})

		It("should delete failed when tower is out of service", func() {
			mockNewVMService = func(_ goctx.Context, _ infrav1.Tower, _ logr.Logger) (service.VMService, error) {
				return mockVMService, errors.New("get vm service failed")
			}
			ctrlMgrContext := fake.NewControllerManagerContext(elfCluster, cluster)
			ctrlContext := &context.ControllerContext{
				ControllerManagerContext: ctrlMgrContext,
				Logger:                   ctrllog.Log,
			}
			fake.InitClusterOwnerReferences(ctrlContext, elfCluster, cluster)

			reconciler := &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			elfClusterKey := capiutil.ObjectKey(elfCluster)
			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: elfClusterKey})
			Expect(result).To(BeZero())
			Expect(err).To(HaveOccurred())
			Expect(reconciler.Client.Get(reconciler, elfClusterKey, elfCluster)).To(Succeed())
			Expect(elfCluster.Finalizers).To(ContainElement(infrav1.ClusterFinalizer))
		})

		It("should force delete when tower is out of service and cluster need to force delete", func() {
			mockNewVMService = func(_ goctx.Context, _ infrav1.Tower, _ logr.Logger) (service.VMService, error) {
				return mockVMService, errors.New("get vm service failed")
			}
			elfCluster.Annotations = map[string]string{
				infrav1.ElfClusterForceDeleteAnnotation: "",
			}
			ctrlMgrContext := fake.NewControllerManagerContext(elfCluster, cluster)
			ctrlContext := &context.ControllerContext{
				ControllerManagerContext: ctrlMgrContext,
				Logger:                   ctrllog.Log,
			}
			fake.InitClusterOwnerReferences(ctrlContext, elfCluster, cluster)

			reconciler := &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			elfClusterKey := capiutil.ObjectKey(elfCluster)
			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: elfClusterKey})
			Expect(result).To(BeZero())
			Expect(err).ToNot(HaveOccurred())
			Expect(logBuffer.String()).To(ContainSubstring(""))
			Expect(apierrors.IsNotFound(reconciler.Client.Get(reconciler, elfClusterKey, elfCluster))).To(BeTrue())
		})

		It("should delete the duplicate virtual machines", func() {
			vm1 := fake.NewTowerVM()
			vm2 := fake.NewTowerVM()
			vm2.EntityAsyncStatus = nil
			vm2.Status = models.NewVMStatus(models.VMStatusSTOPPED)
			task := fake.NewTowerTask()
			task.Status = models.NewTaskStatus(models.TaskStatusSUCCESSED)
			ctrlMgrContext := fake.NewControllerManagerContext(elfCluster, cluster)
			ctrlContext := &context.ControllerContext{
				ControllerManagerContext: ctrlMgrContext,
				Logger:                   ctrllog.Log,
			}
			fake.InitClusterOwnerReferences(ctrlContext, elfCluster, cluster)

			mockVMService.EXPECT().DeleteVMPlacementGroupsByName(towerresources.GetVMPlacementGroupNamePrefix(cluster)).Return(nil)
			mockVMService.EXPECT().DeleteLabel(towerresources.GetVMLabelClusterName(), elfCluster.Name, true).Return("labelid", nil)
			mockVMService.EXPECT().DeleteLabel(towerresources.GetVMLabelVIP(), elfCluster.Spec.ControlPlaneEndpoint.Host, false).Return("labelid", nil)
			mockVMService.EXPECT().DeleteLabel(towerresources.GetVMLabelNamespace(), elfCluster.Namespace, true).Return("", nil)
			mockVMService.EXPECT().DeleteLabel(towerresources.GetVMLabelManaged(), "true", true).Return("", nil)
			mockVMService.EXPECT().FindVMsByName(cluster.Name).Return([]*models.VM{vm1, vm2}, nil)
			mockVMService.EXPECT().Delete(*vm2.ID).Return(task, nil)
			mockVMService.EXPECT().WaitTask(*task.ID, waitDuplicateVMTaskTimeout, waitDuplicateVMTaskInterval).Return(task, nil)

			reconciler := &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			elfClusterKey := capiutil.ObjectKey(elfCluster)
			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: elfClusterKey})
			Expect(logBuffer.String()).To(ContainSubstring("Waiting for VM task done before deleting"))
			Expect(logBuffer.String()).To(ContainSubstring("Duplicate VM already deleted"))
			Expect(result.RequeueAfter).To(Equal(duplicateVMDefaultRequeueTimeout))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(reconciler.Client.Get(reconciler, elfClusterKey, elfCluster)).To(Succeed())
			Expect(elfCluster.Finalizers).To(ContainElement(infrav1.ClusterFinalizer))
		})
	})

	Context("deleteVMs", func() {
		BeforeEach(func() {
			ctrlutil.AddFinalizer(elfCluster, infrav1.ClusterFinalizer)
			elfCluster.DeletionTimestamp = &metav1.Time{Time: time.Now().UTC()}
		})

		It("should do nothing without a virtual machine", func() {
			ctrlMgrContext := fake.NewControllerManagerContext(elfCluster, cluster)
			ctrlContext := &context.ControllerContext{
				ControllerManagerContext: ctrlMgrContext,
				Logger:                   ctrllog.Log,
			}
			clusterContext := newClustereContext(ctrlContext, elfCluster, cluster, mockVMService)
			reconciler := &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			result, err := reconciler.deleteVMs(clusterContext, nil)
			Expect(result).To(BeZero())
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should not delete the VM that in operation", func() {
			vm := fake.NewTowerVM()
			task := fake.NewTowerTask()
			task.Status = models.NewTaskStatus(models.TaskStatusSUCCESSED)

			ctrlMgrContext := fake.NewControllerManagerContext(elfCluster, cluster)
			ctrlContext := &context.ControllerContext{
				ControllerManagerContext: ctrlMgrContext,
				Logger:                   ctrllog.Log,
			}
			clusterContext := newClustereContext(ctrlContext, elfCluster, cluster, mockVMService)
			reconciler := &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			result, err := reconciler.deleteVMs(clusterContext, []*models.VM{vm})
			Expect(result.RequeueAfter).To(Equal(duplicateVMDefaultRequeueTimeout))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(logBuffer.String()).To(ContainSubstring("Waiting for VM task done before deleting"))
		})

		It("should power off the VM and then delete it when it is running", func() {
			vm := fake.NewTowerVM()
			vm.EntityAsyncStatus = nil
			vm.Status = models.NewVMStatus(models.VMStatusRUNNING)
			task := fake.NewTowerTask()
			task.Status = models.NewTaskStatus(models.TaskStatusSUCCESSED)
			mockVMService.EXPECT().PowerOff(*vm.ID).Return(task, nil)
			mockVMService.EXPECT().Delete(*vm.ID).Return(task, nil)
			mockVMService.EXPECT().WaitTask(*task.ID, waitDuplicateVMTaskTimeout, waitDuplicateVMTaskInterval).Times(2).Return(task, nil)

			ctrlMgrContext := fake.NewControllerManagerContext(elfCluster, cluster)
			ctrlContext := &context.ControllerContext{
				ControllerManagerContext: ctrlMgrContext,
				Logger:                   ctrllog.Log,
			}
			clusterContext := newClustereContext(ctrlContext, elfCluster, cluster, mockVMService)
			reconciler := &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			result, err := reconciler.deleteVMs(clusterContext, []*models.VM{vm})
			Expect(result).To(BeZero())
			Expect(err).ShouldNot(HaveOccurred())
			Expect(logBuffer.String()).To(ContainSubstring("Duplicate VM already deleted"))
		})

		It("should return an error when the task fail or times out", func() {
			vm := fake.NewTowerVM()
			vm.EntityAsyncStatus = nil
			vm.Status = models.NewVMStatus(models.VMStatusSTOPPED)
			task := fake.NewTowerTask()
			task.Status = models.NewTaskStatus(models.TaskStatusSUCCESSED)
			mockVMService.EXPECT().Delete(*vm.ID).Return(task, nil)
			mockVMService.EXPECT().WaitTask(*task.ID, waitDuplicateVMTaskTimeout, waitDuplicateVMTaskInterval).Return(nil, errors.New("some error"))

			ctrlMgrContext := fake.NewControllerManagerContext(elfCluster, cluster)
			ctrlContext := &context.ControllerContext{
				ControllerManagerContext: ctrlMgrContext,
				Logger:                   ctrllog.Log,
			}
			clusterContext := newClustereContext(ctrlContext, elfCluster, cluster, mockVMService)
			reconciler := &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			result, err := reconciler.deleteVMs(clusterContext, []*models.VM{vm})
			Expect(result).To(BeZero())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("failed to wait for VM deletion task done in %s: key %s/%s, taskID %s", waitDuplicateVMTaskTimeout, *vm.Name, *vm.ID, *task.ID)))

			task.Status = models.NewTaskStatus(models.TaskStatusFAILED)
			mockVMService.EXPECT().Delete(*vm.ID).Return(task, nil)
			mockVMService.EXPECT().WaitTask(*task.ID, waitDuplicateVMTaskTimeout, waitDuplicateVMTaskInterval).Return(task, nil)
			clusterContext = newClustereContext(ctrlContext, elfCluster, cluster, mockVMService)
			reconciler = &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			result, err = reconciler.deleteVMs(clusterContext, []*models.VM{vm})
			Expect(result).To(BeZero())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("failed to delete VM %s/%s in task %s", *vm.Name, *vm.ID, *task.ID)))

			vm.Status = models.NewVMStatus(models.VMStatusRUNNING)
			task.Status = models.NewTaskStatus(models.TaskStatusSUCCESSED)
			mockVMService.EXPECT().PowerOff(*vm.ID).Return(task, nil)
			mockVMService.EXPECT().WaitTask(*task.ID, waitDuplicateVMTaskTimeout, waitDuplicateVMTaskInterval).Return(nil, errors.New("some error"))
			reconciler = &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			result, err = reconciler.deleteVMs(clusterContext, []*models.VM{vm})
			Expect(result).To(BeZero())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("failed to wait for VM power off task done in %s: key %s/%s, taskID %s", waitDuplicateVMTaskTimeout, *vm.Name, *vm.ID, *task.ID)))

			task.Status = models.NewTaskStatus(models.TaskStatusFAILED)
			mockVMService.EXPECT().PowerOff(*vm.ID).Return(task, nil)
			mockVMService.EXPECT().WaitTask(*task.ID, waitDuplicateVMTaskTimeout, waitDuplicateVMTaskInterval).Return(task, nil)
			clusterContext = newClustereContext(ctrlContext, elfCluster, cluster, mockVMService)
			reconciler = &ElfClusterReconciler{ControllerContext: ctrlContext, NewVMService: mockNewVMService}
			result, err = reconciler.deleteVMs(clusterContext, []*models.VM{vm})
			Expect(result).To(BeZero())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("failed to power off VM %s/%s in task %s", *vm.Name, *vm.ID, *task.ID)))
		})
	})
})

func newClustereContext(ctrlCtx *context.ControllerContext, elfCluster *infrav1.ElfCluster, cluster *clusterv1.Cluster, vmService service.VMService) *context.ClusterContext {
	return &context.ClusterContext{
		ControllerContext: ctrlCtx,
		Cluster:           cluster,
		ElfCluster:        elfCluster,
		Logger:            ctrlCtx.Logger,
		VMService:         vmService,
	}
}

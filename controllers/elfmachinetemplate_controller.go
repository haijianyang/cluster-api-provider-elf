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
	goctx "context"
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apitypes "k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	capiutil "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "github.com/smartxworks/cluster-api-provider-elf/api/v1beta1"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/config"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/context"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/service"
	machineutil "github.com/smartxworks/cluster-api-provider-elf/pkg/util/machine"
)

// ElfMachineTemplateReconciler reconciles a ElfMachineTemplate object.
type ElfMachineTemplateReconciler struct {
	*context.ControllerContext
	NewVMService service.NewVMServiceFunc
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=elfmachinetemplates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=elfmachinetemplates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=elfmachinetemplates/finalizers,verbs=update

// AddMachineTemplateControllerToManager adds the ElfMachineTemplate controller to the provided
// manager.
func AddMachineTemplateControllerToManager(ctx *context.ControllerManagerContext, mgr ctrlmgr.Manager, options controller.Options) error {
	var (
		controlledType     = &infrav1.ElfMachineTemplate{}
		controlledTypeName = reflect.TypeOf(controlledType).Elem().Name()

		controllerNameShort = fmt.Sprintf("%s-controller", strings.ToLower(controlledTypeName))
	)

	// Build the controller context.
	controllerContext := &context.ControllerContext{
		ControllerManagerContext: ctx,
		Name:                     controllerNameShort,
		Logger:                   ctx.Logger.WithName(controllerNameShort),
	}

	reconciler := &ElfMachineTemplateReconciler{
		ControllerContext: controllerContext,
		NewVMService:      service.NewVMService,
	}

	return ctrl.NewControllerManagedBy(mgr).
		// Watch the controlled, infrastructure resource.
		For(controlledType).
		WithOptions(options).
		// WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(ctrl.LoggerFrom(ctx), ctx.WatchFilterValue)).
		Complete(reconciler)
}

func (r *ElfMachineTemplateReconciler) Reconcile(ctx goctx.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	// Get the ElfMachineTemplate resource for this request.
	var elfMachineTemplate infrav1.ElfMachineTemplate
	if err := r.Client.Get(r, req.NamespacedName, &elfMachineTemplate); err != nil {
		if apierrors.IsNotFound(err) {
			r.Logger.Info("ElfMachineTemplate not found, won't reconcile", "key", req.NamespacedName)

			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	// Fetch the CAPI Cluster.
	cluster, err := capiutil.GetOwnerCluster(r, r.Client, elfMachineTemplate.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if cluster == nil {
		r.Logger.Info("Waiting for Cluster Controller to set OwnerRef on ElfMachineTemplate",
			"namespace", elfMachineTemplate.Namespace, "elfCluster", elfMachineTemplate.Name)

		return reconcile.Result{}, nil
	}

	if annotations.IsPaused(cluster, &elfMachineTemplate) {
		r.Logger.V(4).Info("ElfMachineTemplate linked to a cluster that is paused",
			"namespace", elfMachineTemplate.Namespace, "elfMachineTemplate", elfMachineTemplate.Name)

		return reconcile.Result{}, nil
	}

	// Fetch the ElfCluster
	var elfCluster infrav1.ElfCluster
	if err := r.Client.Get(r, client.ObjectKey{
		Namespace: cluster.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}, &elfCluster); err != nil {
		if apierrors.IsNotFound(err) {
			r.Logger.Info("ElfMachine Waiting for ElfCluster",
				"namespace", elfMachineTemplate.Namespace, "elfMachineTemplate", elfMachineTemplate.Name)
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	logger := r.Logger.WithValues("namespace", elfMachineTemplate.Namespace,
		"elfCluster", elfCluster.Name, "elfMachineTemplate", elfMachineTemplate.Name)

	// Create the machine context for this request.
	machineTemplateContext := &context.MachineTemplateContext{
		ControllerContext:  r.ControllerContext,
		Cluster:            cluster,
		ElfCluster:         &elfCluster,
		ElfMachineTemplate: &elfMachineTemplate,
		Logger:             logger,
	}

	if elfMachineTemplate.ObjectMeta.DeletionTimestamp.IsZero() || !elfCluster.HasForceDeleteCluster() {
		vmService, err := r.NewVMService(r.Context, elfCluster.GetTower(), logger)
		if err != nil {
			return reconcile.Result{}, err
		}

		machineTemplateContext.VMService = vmService
	}

	// Handle deleted machines
	if !elfMachineTemplate.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// Handle non-deleted machines
	return r.reconcileNormal(machineTemplateContext)
}

func (r *ElfMachineTemplateReconciler) reconcileNormal(ctx *context.MachineTemplateContext) (reconcile.Result, error) {
	return r.reconcileMachineResources(ctx)
}

// reconcileMachineResources ensures that the resources(disk capacity) of the
// virtual machines are the same as expected by ElfMachine.
// TODO: CPU and memory will be supported in the future.
func (r *ElfMachineTemplateReconciler) reconcileMachineResources(ctx *context.MachineTemplateContext) (reconcile.Result, error) {
	if ok, err := r.reconcileCPResources(ctx); err != nil {
		return reconcile.Result{}, err
	} else if !ok {
		return reconcile.Result{RequeueAfter: config.DefaultRequeueTimeout}, nil
	}

	if ok, err := r.reconcileWorkerResources(ctx); err != nil {
		return reconcile.Result{}, err
	} else if !ok {
		return reconcile.Result{RequeueAfter: config.DefaultRequeueTimeout}, nil
	}

	return reconcile.Result{}, nil
}

// reconcileMachineResources ensures that the resources(disk capacity) of the
// control plane virtual machines are the same as expected by ElfMachine.
func (r *ElfMachineTemplateReconciler) reconcileCPResources(ctx *context.MachineTemplateContext) (bool, error) {
	var kcp controlplanev1.KubeadmControlPlane
	if err := ctx.Client.Get(ctx, apitypes.NamespacedName{
		Namespace: ctx.Cluster.Spec.ControlPlaneRef.Namespace,
		Name:      ctx.Cluster.Spec.ControlPlaneRef.Name,
	}, &kcp); err != nil {
		return false, err
	}

	if kcp.Spec.MachineTemplate.InfrastructureRef.Namespace != ctx.ElfMachineTemplate.Namespace ||
		kcp.Spec.MachineTemplate.InfrastructureRef.Name != ctx.ElfMachineTemplate.Name {
		return true, nil
	}

	// TODO: 需要优化成筛选需要进行磁盘扩容的 elfMachines，
	// 例如在滚动更新过程，旧的节点不需要金小磁盘扩容。
	elfMachines, err := machineutil.GetControlPlaneElfMachinesInCluster(ctx, ctx.Client, ctx.Cluster.Namespace, ctx.Cluster.Name)
	if err != nil {
		return false, err
	}

	// TODO: 检查是否满足磁盘扩容条件，例如有节点故障等。

	updatingResourcesElfMachines, needUpdatedResourceElfMachines := selectNotUpToDateElfMachines(ctx.ElfMachineTemplate, elfMachines)
	if len(updatingResourcesElfMachines) == 0 && len(needUpdatedResourceElfMachines) == 0 {
		return true, nil
	}

	// Only one CP ElfMachine is allowed to update resources at the same time.
	if len(updatingResourcesElfMachines) > 0 {
		ctx.Logger.V(1).Info("Waiting for control plane ElfMachines to be updated resources", "updatingCount", len(updatingResourcesElfMachines), "needUpdatedCount", len(needUpdatedResourceElfMachines))

		return false, nil
	}

	toBeUpdatedElfMachine := needUpdatedResourceElfMachines[0]
	if err := markElfMachineToBeUpdatedResources(ctx, ctx.ElfMachineTemplate, toBeUpdatedElfMachine); err != nil {
		return false, err
	}

	return false, err
}

// reconcileMachineResources ensures that the resources(disk capacity) of the
// worker virtual machines are the same as expected by ElfMachine.
func (r *ElfMachineTemplateReconciler) reconcileWorkerResources(ctx *context.MachineTemplateContext) (bool, error) {
	mds, err := machineutil.GetMDsForCluster(ctx, ctx.Client, ctx.Cluster.Namespace, ctx.Cluster.Name)
	if err != nil {
		return false, err
	}

	allElfMachinesUpToDate := true
	for i := 0; i < len(mds); i++ {
		if ctx.ElfMachineTemplate.Name != mds[i].Spec.Template.Spec.InfrastructureRef.Name {
			continue
		}

		if ok, err := r.reconcileWorkerResourcesForMD(ctx, mds[i]); err != nil {
			return false, err
		} else if !ok {
			allElfMachinesUpToDate = false
		}
	}

	return allElfMachinesUpToDate, nil
}

// reconcileWorkerResourcesForMD ensures that the resources(disk capacity) of the
// worker virtual machines managed by the md are the same as expected by ElfMachine.
func (r *ElfMachineTemplateReconciler) reconcileWorkerResourcesForMD(ctx *context.MachineTemplateContext, md *clusterv1.MachineDeployment) (bool, error) {
	// TODO: 应优化成筛选出需要扩容的磁盘，根据最新的 MS 筛选，但目前没有办法找到最新的 MS
	elfMachines, err := machineutil.GetElfMachinesForMD(ctx, ctx.Client, ctx.Cluster, md)
	if err != nil {
		return false, err
	}

	// TODO: 1.检查是否满足磁盘扩容条件，例如有节点故障等；2.控制并发扩容的 elfMachines 数量

	updatingResourcesElfMachines, needUpdatedResourcesElfMachines := selectNotUpToDateElfMachines(ctx.ElfMachineTemplate, elfMachines)
	for i := 0; i < len(needUpdatedResourcesElfMachines); i++ {
		toBeUpdatedElfMachine := needUpdatedResourcesElfMachines[i]
		if err := markElfMachineToBeUpdatedResources(ctx, ctx.ElfMachineTemplate, toBeUpdatedElfMachine); err != nil {
			return false, err
		}
	}

	notUpToDateElfMachineCount := len(updatingResourcesElfMachines) + len(needUpdatedResourcesElfMachines)
	if notUpToDateElfMachineCount > 0 {
		ctx.Logger.V(2).Info("Waiting for worker ElfMachines to be updated resources", "md", md.Name, "count", notUpToDateElfMachineCount)

		return false, nil
	}

	return true, nil
}

func selectNotUpToDateElfMachines(elfMachineTemplate *infrav1.ElfMachineTemplate, elfMachines []*infrav1.ElfMachine) ([]*infrav1.ElfMachine, []*infrav1.ElfMachine) {
	var updatingResourcesElfMachines []*infrav1.ElfMachine
	var needUpdatedResourcesElfMachines []*infrav1.ElfMachine
	for i := 0; i < len(elfMachines); i++ {
		elfMachine := elfMachines[i]
		if machineutil.IsUpdatingElfMachineResources(elfMachine) {
			updatingResourcesElfMachines = append(updatingResourcesElfMachines, elfMachine)
			continue
		}

		if machineutil.NeedUpdateElfMachineResources(elfMachineTemplate, elfMachine) {
			needUpdatedResourcesElfMachines = append(needUpdatedResourcesElfMachines, elfMachine)
		}
	}

	return updatingResourcesElfMachines, needUpdatedResourcesElfMachines
}

// markElfMachineToBeUpdatedResources synchronizes the expected resource values
// from the ElfMachineTemplate and marks the ElfMachine to be updated resources.
func markElfMachineToBeUpdatedResources(ctx *context.MachineTemplateContext, elfMachineTemplate *infrav1.ElfMachineTemplate, elfMachine *infrav1.ElfMachine) error {
	patchHelper, err := patch.NewHelper(elfMachine, ctx.Client)
	if err != nil {
		return err
	}

	// Ensure resources are up to date.
	diskGiB := elfMachineTemplate.Spec.Template.Spec.DiskGiB
	elfMachine.Spec.DiskGiB = elfMachineTemplate.Spec.Template.Spec.DiskGiB
	conditions.MarkFalse(elfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.WaitingForResourcesHotUpdateReason, clusterv1.ConditionSeverityInfo, "")

	ctx.Logger.Info(fmt.Sprintf("Resources of ElfMachine is not up to date, marking for updating resources(disk: %d -> %d)", diskGiB, elfMachine.Spec.DiskGiB), "elfMachine", elfMachine.Name)

	if err := patchHelper.Patch(ctx, elfMachine); err != nil {
		return errors.Wrapf(err, "failed to patch ElfMachine %s to mark for updating resources", elfMachine.Name)
	}

	return nil
}

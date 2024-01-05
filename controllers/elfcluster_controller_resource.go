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

package controllers

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/collections"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "github.com/smartxworks/cluster-api-provider-elf/api/v1beta1"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/config"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/context"
	machineutil "github.com/smartxworks/cluster-api-provider-elf/pkg/util/machine"
)

func (r *ElfClusterReconciler) reconcileMachineResources(ctx *context.ClusterContext) (reconcile.Result, error) {
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

func (r *ElfClusterReconciler) preflightChecks(ctx *context.ClusterContext) (bool, error) {
	cpMachines, err := machineutil.GetControlPlaneMachinesForCluster(ctx, ctx.Client, ctx.Cluster)
	if err != nil {
		return false, err
	}

	machines := collections.FromMachines(cpMachines...)
	deletingMachines := machines.Filter(collections.HasDeletionTimestamp)
	if len(deletingMachines) > 0 {
		ctx.Logger.Info("Waiting for machines to be deleted", "machines", deletingMachines.Names())

		return false, nil
	}

	allMachineHealthConditions := []clusterv1.ConditionType{
		controlplanev1.MachineAPIServerPodHealthyCondition,
		controlplanev1.MachineControllerManagerPodHealthyCondition,
		controlplanev1.MachineSchedulerPodHealthyCondition,
		controlplanev1.MachineEtcdPodHealthyCondition,
		controlplanev1.MachineEtcdMemberHealthyCondition,
	}
	machineErrors := []error{}
	for _, machine := range machines {
		if machine.Status.NodeRef == nil {
			// The conditions will only ever be set on a Machine if we're able to correlate a Machine to a Node.
			// Correlating Machines to Nodes requires the nodeRef to be set.
			// Instead of confusing users with errors about that the conditions are not set, let's point them
			// towards the unset nodeRef (which is the root cause of the conditions not being there).
			machineErrors = append(machineErrors, errors.Errorf("Machine %s does not have a corresponding Node yet (Machine.status.nodeRef not set)", machine.Name))
		} else {
			for _, condition := range allMachineHealthConditions {
				if err := preflightCheckCondition("Machine", machine, condition); err != nil {
					machineErrors = append(machineErrors, err)
				}
			}
		}
	}

	if len(machineErrors) > 0 {
		aggregatedError := kerrors.NewAggregate(machineErrors)
		ctx.Logger.Info("Waiting for control plane to pass preflight checks", "failures", aggregatedError.Error())

		return false, nil
	}

	return true, nil
}

func preflightCheckCondition(kind string, obj conditions.Getter, condition clusterv1.ConditionType) error {
	c := conditions.Get(obj, condition)
	if c == nil {
		return errors.Errorf("%s %s does not have %s condition", kind, obj.GetName(), condition)
	}
	if c.Status == corev1.ConditionFalse {
		return errors.Errorf("%s %s reports %s condition is false (%s, %s)", kind, obj.GetName(), condition, c.Severity, c.Message)
	}
	if c.Status == corev1.ConditionUnknown {
		return errors.Errorf("%s %s reports %s condition is unknown (%s)", kind, obj.GetName(), condition, c.Message)
	}
	return nil
}

func (r *ElfClusterReconciler) reconcileCPResources(ctx *context.ClusterContext) (bool, error) {
	var kcp controlplanev1.KubeadmControlPlane
	if err := ctx.Client.Get(ctx, apitypes.NamespacedName{Namespace: ctx.Cluster.Spec.ControlPlaneRef.Namespace, Name: ctx.Cluster.Spec.ControlPlaneRef.Name}, &kcp); err != nil {
		return false, err
	}

	var elfMachineTemplate infrav1.ElfMachineTemplate
	if err := ctx.Client.Get(ctx, apitypes.NamespacedName{Namespace: kcp.Spec.MachineTemplate.InfrastructureRef.Namespace, Name: kcp.Spec.MachineTemplate.InfrastructureRef.Name}, &elfMachineTemplate); err != nil {
		return false, err
	}

	elfMachines, err := machineutil.GetControlPlaneElfMachinesInCluster(ctx, ctx.Client, ctx.Cluster.Namespace, ctx.Cluster.Name)
	if err != nil {
		return false, err
	}

	updatingResourceElfMachines, needUpdatedResourceElfMachines := selectNotUpToDateElfMachines(&elfMachineTemplate, elfMachines)
	if len(updatingResourceElfMachines) == 0 && len(needUpdatedResourceElfMachines) == 0 {
		return true, nil
	}

	if len(updatingResourceElfMachines) > 0 {
		ctx.Logger.V(2).Info("Waiting for control plane ElfMachines to be updated resources", "updatingCount", len(updatingResourceElfMachines), "needUpdatedCount", len(needUpdatedResourceElfMachines))

		return false, nil
	}

	if ok, err := r.preflightChecks(ctx); err != nil || !ok {
		return ok, err
	}

	toBeUpdatedElfMachine := needUpdatedResourceElfMachines[0]
	if err := markElfMachineToBeUpdatedResources(ctx, &elfMachineTemplate, toBeUpdatedElfMachine); err != nil {
		return false, err
	}

	return false, err
}

func (r *ElfClusterReconciler) reconcileWorkerResources(ctx *context.ClusterContext) (bool, error) {
	mds, err := machineutil.GetMDsForCluster(ctx, ctx.Client, ctx.Cluster.Namespace, ctx.Cluster.Name)
	if err != nil {
		return false, err
	}

	hasNotUpToDateElfMachine := false
	for i := 0; i < len(mds); i++ {
		md := mds[i]
		elfMachineTemplateRef := md.Spec.Template.Spec.InfrastructureRef

		var elfMachineTemplate infrav1.ElfMachineTemplate
		namespace := elfMachineTemplateRef.Namespace
		if namespace == "" {
			namespace = md.Namespace
		}
		if err := ctx.Client.Get(ctx, apitypes.NamespacedName{Namespace: namespace, Name: elfMachineTemplateRef.Name}, &elfMachineTemplate); err != nil {
			return false, err
		}

		elfMachines, err := machineutil.GetElfMachinesForMD(ctx, ctx.Client, ctx.Cluster, md)
		if err != nil {
			return false, err
		}

		updatingResourceElfMachines, needUpdatedResourceElfMachines := selectNotUpToDateElfMachines(&elfMachineTemplate, elfMachines)
		for i := 0; i < len(needUpdatedResourceElfMachines); i++ {
			toBeUpdatedElfMachine := needUpdatedResourceElfMachines[i]
			if err := markElfMachineToBeUpdatedResources(ctx, &elfMachineTemplate, toBeUpdatedElfMachine); err != nil {
				return false, err
			}
		}

		count := len(updatingResourceElfMachines) + len(needUpdatedResourceElfMachines)
		if count > 0 {
			hasNotUpToDateElfMachine = true

			ctx.Logger.V(2).Info("Waiting for worker ElfMachines to be updated resources", "md", md.Name, "count", count)
		}
	}

	if hasNotUpToDateElfMachine {
		return false, nil
	}

	return true, nil
}

func markElfMachineToBeUpdatedResources(ctx *context.ClusterContext, elfMachineTemplate *infrav1.ElfMachineTemplate, elfMachine *infrav1.ElfMachine) error {
	patchHelper, err := patch.NewHelper(elfMachine, ctx.Client)
	if err != nil {
		return err
	}

	// Ensure resources are up to date.
	elfMachine.Spec.NumCPUs = elfMachineTemplate.Spec.Template.Spec.NumCPUs
	elfMachine.Spec.MemoryMiB = elfMachineTemplate.Spec.Template.Spec.MemoryMiB

	conditions.MarkFalse(elfMachine, infrav1.ResourceHotUpdatedCondition, infrav1.WaitingForResourceHotUpdateReason, clusterv1.ConditionSeverityInfo, "")

	ctx.Logger.Info("Resources of ElfMachine is not up to date, marking for updating resources", "elfMachine", elfMachine.Name)

	if err := patchHelper.Patch(ctx, elfMachine); err != nil {
		return errors.Wrapf(err, "failed to patch ElfMachine %s to mark for updating resources", elfMachine.Name)
	}

	return nil
}

func selectNotUpToDateElfMachines(elfMachineTemplate *infrav1.ElfMachineTemplate, elfMachines []*infrav1.ElfMachine) ([]*infrav1.ElfMachine, []*infrav1.ElfMachine) {
	var updatingResourceElfMachines []*infrav1.ElfMachine
	var needUpdatedResourceElfMachines []*infrav1.ElfMachine
	for i := 0; i < len(elfMachines); i++ {
		if machineutil.IsUpdatingElfMachineResources(elfMachines[i]) {
			updatingResourceElfMachines = append(updatingResourceElfMachines, elfMachines[i])
		} else if machineutil.NeedUpdateElfMachineResources(elfMachineTemplate, elfMachines[i]) {
			needUpdatedResourceElfMachines = append(needUpdatedResourceElfMachines, elfMachines[i])
		}
	}

	return updatingResourceElfMachines, needUpdatedResourceElfMachines
}

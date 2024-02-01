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
	"fmt"

	_ "embed"

	"github.com/pkg/errors"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capiremote "sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentv1 "github.com/smartxworks/cluster-api-provider-elf/api/v1alpha1"
	infrav1 "github.com/smartxworks/cluster-api-provider-elf/api/v1beta1"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/context"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/hostagent"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/service"
	annotationsutil "github.com/smartxworks/cluster-api-provider-elf/pkg/util/annotations"
	machineutil "github.com/smartxworks/cluster-api-provider-elf/pkg/util/machine"
)

func (r *ElfMachineReconciler) reconcileVMResources2(ctx *context.MachineContext, vm *models.VM) (bool, error) {
	if !machineutil.IsUpdatingElfMachineResources(ctx.ElfMachine) {
		return true, nil
	}

	if !service.IsVMResourcesUpToDate(ctx.ElfMachine, vm) {
		return false, r.updateVMResources(ctx, vm)
	}

	kubeClient, err := capiremote.NewClusterClient(ctx, "", ctx.Client, client.ObjectKey{Namespace: ctx.Cluster.Namespace, Name: ctx.Cluster.Name})
	if err != nil {
		return false, err
	}

	var restartKubeletJob *agentv1.HostOperationJob
	agentJobName := annotationsutil.HostAgentJobName(ctx.ElfMachine)
	if agentJobName != "" {
		restartKubeletJob, err = hostagent.GetHostJob(ctx, kubeClient, ctx.ElfMachine.Namespace, agentJobName)
		if err != nil && !apierrors.IsNotFound(err) {
			return false, err
		}
	}
	if restartKubeletJob == nil {
		restartKubeletJob, err = hostagent.RestartMachineKubelet(ctx, kubeClient, ctx.ElfMachine)
		if err != nil {
			conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.RestartingKubeletFailedReason, clusterv1.ConditionSeverityInfo, err.Error())

			return false, err
		}

		annotationsutil.AddAnnotations(ctx.ElfMachine, map[string]string{infrav1.HostAgentJobNameAnnotation: restartKubeletJob.Name})

		conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.RestartingKubeletReason, clusterv1.ConditionSeverityInfo, "")

		ctx.Logger.Info("Waiting for kubelet to be restarted", "hostAgentJob", restartKubeletJob.Name)

		return false, nil
	}

	switch restartKubeletJob.Status.Phase {
	case agentv1.PhaseSucceeded:
		annotationsutil.RemoveAnnotation(ctx.ElfMachine, infrav1.HostAgentJobNameAnnotation)
		conditions.MarkTrue(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition)
		ctx.Logger.Info("Restart kubelet succeeded", "hostAgentJob", restartKubeletJob.Name)
	case agentv1.PhaseFailed:
		annotationsutil.RemoveAnnotation(ctx.ElfMachine, infrav1.HostAgentJobNameAnnotation)
		conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.RestartingKubeletFailedReason, clusterv1.ConditionSeverityWarning, restartKubeletJob.Status.FailureMessage)
		ctx.Logger.Info("Restart kubelet failed, will try again", "hostAgentJob", restartKubeletJob.Name)

		return false, nil
	default:
		ctx.Logger.Info("Waiting for restart kubelet job done", "jobStatus", restartKubeletJob.Status.Phase)

		return false, nil
	}

	return true, nil
}

func (r *ElfMachineReconciler) updateVMResources(ctx *context.MachineContext, vm *models.VM) (bool, error) {
	if !service.IsVMResourcesUpToDate(ctx.ElfMachine, vm) {
		return true, nil
	}

	if ok := acquireTicketForUpdatingVM(ctx.ElfMachine.Name); !ok {
		ctx.Logger.V(1).Info(fmt.Sprintf("The VM operation reaches rate limit, skip updating VM %s resources", ctx.ElfMachine.Status.VMRef))

		return false, nil
	}

	withTaskVM, err := ctx.VMService.UpdateVM(vm, ctx.ElfMachine)
	if err != nil {
		conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.ExpandingVMResourcesFailedReason, clusterv1.ConditionSeverityWarning, err.Error())

		return false, errors.Wrapf(err, "failed to trigger update resources for VM %s", ctx)
	}

	conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.ExpandingVMResourcesReason, clusterv1.ConditionSeverityInfo, "")

	ctx.ElfMachine.SetTask(*withTaskVM.TaskID)

	ctx.Logger.Info("Waiting for the VM to be updated resources", "vmRef", ctx.ElfMachine.Status.VMRef, "taskRef", ctx.ElfMachine.Status.TaskRef)

	return false, nil
}

func (r *ElfMachineReconciler) restartKubelet(ctx *context.MachineContext, vm *models.VM) (bool, error) {
	reason := conditions.GetReason(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition)
	if reason == "" {
		return true, nil
	} else if reason != infrav1.ExpandingVMResourcesReason &&
		reason != infrav1.RestartingKubeletReason &&
		reason != infrav1.RestartingKubeletFailedReason {
		return true, nil
	}

	kubeClient, err := capiremote.NewClusterClient(ctx, "", ctx.Client, client.ObjectKey{Namespace: ctx.Cluster.Namespace, Name: ctx.Cluster.Name})
	if err != nil {
		return false, err
	}

	var restartKubeletJob *agentv1.HostOperationJob
	agentJobName := annotationsutil.HostAgentJobName(ctx.ElfMachine)
	if agentJobName != "" {
		restartKubeletJob, err = hostagent.GetHostJob(ctx, kubeClient, ctx.ElfMachine.Namespace, agentJobName)
		if err != nil && !apierrors.IsNotFound(err) {
			return false, err
		}
	}
	if restartKubeletJob == nil {
		restartKubeletJob, err = hostagent.RestartMachineKubelet(ctx, kubeClient, ctx.ElfMachine)
		if err != nil {
			conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.RestartingKubeletFailedReason, clusterv1.ConditionSeverityInfo, err.Error())

			return false, err
		}

		annotationsutil.AddAnnotations(ctx.ElfMachine, map[string]string{infrav1.HostAgentJobNameAnnotation: restartKubeletJob.Name})

		conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.RestartingKubeletReason, clusterv1.ConditionSeverityInfo, "")

		ctx.Logger.Info("Waiting for kubelet to be restarted", "hostAgentJob", restartKubeletJob.Name)

		return false, nil
	}

	switch restartKubeletJob.Status.Phase {
	case agentv1.PhaseSucceeded:
		annotationsutil.RemoveAnnotation(ctx.ElfMachine, infrav1.HostAgentJobNameAnnotation)
		conditions.MarkTrue(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition)
		ctx.Logger.Info("Restart kubelet succeeded", "hostAgentJob", restartKubeletJob.Name)
	case agentv1.PhaseFailed:
		annotationsutil.RemoveAnnotation(ctx.ElfMachine, infrav1.HostAgentJobNameAnnotation)
		conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.RestartingKubeletFailedReason, clusterv1.ConditionSeverityWarning, restartKubeletJob.Status.FailureMessage)
		ctx.Logger.Info("Restart kubelet failed, will try again", "hostAgentJob", restartKubeletJob.Name)

		return false, nil
	default:
		ctx.Logger.Info("Waiting for restart kubelet job done", "jobStatus", restartKubeletJob.Status.Phase)

		return false, nil
	}

	return true, nil
}

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
	"fmt"

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

func (r *ElfMachineReconciler) reconcileVMResources(ctx *context.MachineContext, vm *models.VM) (bool, error) {
	if !machineutil.IsUpdatingElfMachineResources(ctx.ElfMachine) {
		return true, nil
	}

	if ok, err := r.reconcieVMVolume(ctx, vm, infrav1.ResourcesHotUpdatedCondition); err != nil || !ok {
		return ok, err
	}

	// Agent needs to wait for the node exists before it can run and execute commands.
	if ctx.Machine.Status.Phase != string(clusterv1.MachinePhaseRunning) {
		ctx.Logger.Info("Waiting for node exists for host agent running", "phase", ctx.Machine.Status.Phase)

		return false, nil
	}

	kubeClient, err := capiremote.NewClusterClient(ctx, "", ctx.Client, client.ObjectKey{Namespace: ctx.Cluster.Namespace, Name: ctx.Cluster.Name})
	if err != nil {
		return false, err
	}

	var agentJob *agentv1.HostOperationJob
	agentJobName := annotationsutil.HostAgentJobName(ctx.ElfMachine)
	if agentJobName != "" {
		agentJob, err = hostagent.GetHostJob(ctx, kubeClient, ctx.ElfMachine.Namespace, agentJobName)
		if err != nil && !apierrors.IsNotFound(err) {
			return false, err
		}
	}
	if agentJob == nil {
		agentJob, err = hostagent.AddNewDiskCapacityToRoot(ctx, kubeClient, ctx.ElfMachine)
		if err != nil {
			conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.AddingNewDiskCapacityToRootFailedReason, clusterv1.ConditionSeverityInfo, err.Error())

			return false, err
		}

		annotationsutil.AddAnnotations(ctx.ElfMachine, map[string]string{infrav1.HostAgentJobNameAnnotation: agentJob.Name})

		conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.AddingNewDiskCapacityToRootReason, clusterv1.ConditionSeverityInfo, "")

		ctx.Logger.Info("Waiting for disk to be added new disk capacity to root", "hostAgentJob", agentJob.Name)

		return false, nil
	}

	switch agentJob.Status.Phase {
	case agentv1.PhaseSucceeded:
		annotationsutil.RemoveAnnotation(ctx.ElfMachine, infrav1.HostAgentJobNameAnnotation)
		conditions.MarkTrue(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition)
		ctx.Logger.Info("Add new disk capacity to root succeeded", "hostAgentJob", agentJob.Name)
	case agentv1.PhaseFailed:
		annotationsutil.RemoveAnnotation(ctx.ElfMachine, infrav1.HostAgentJobNameAnnotation)
		conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.AddingNewDiskCapacityToRootFailedReason, clusterv1.ConditionSeverityWarning, agentJob.Status.FailureMessage)
		ctx.Logger.Info("Add new disk capacity to root failed, will try again", "hostAgentJob", agentJob.Name)

		return false, nil
	default:
		ctx.Logger.Info("Waiting for adding new disk capacity to root job done", "jobStatus", agentJob.Status.Phase)

		return false, nil
	}

	return true, nil
}

// reconcieVMVolume ensures that the vm disk size is as expected.
func (r *ElfMachineReconciler) reconcieVMVolume(ctx *context.MachineContext, vm *models.VM, conditionType clusterv1.ConditionType) (bool, error) {
	vmDiskIDs := make([]string, len(vm.VMDisks))
	for i := 0; i < len(vm.VMDisks); i++ {
		vmDiskIDs[i] = *vm.VMDisks[i].ID
	}

	vmDisks, err := ctx.VMService.GetVMDisks(vmDiskIDs)
	if err != nil {
		return false, errors.Errorf("no disks found for vm %s/%s", *vm.ID, *vm.Name)
	}

	vmVolume, err := ctx.VMService.GetVMVolume(*vmDisks[0].VMVolume.ID)
	if err != nil {
		return false, err
	}

	diskSize := service.TowerDisk(ctx.ElfMachine.Spec.DiskGiB)
	if *diskSize > *vmVolume.Size {
		if service.IsTowerResourcePerformingAnOperation(vmVolume.EntityAsyncStatus) {
			ctx.Logger.Info("Waiting for vm volume task done", "volume", fmt.Sprintf("%s/%s", *vmVolume.ID, *vmVolume.Name))

			return false, nil
		}

		return false, r.resizeVMVolume(ctx, vmVolume, *diskSize, conditionType)
	} else if *diskSize < *vmVolume.Size {
		conditions.MarkTrue(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition)
		ctx.Logger.Info(fmt.Sprintf("Current disk capacity is larger than expected, skipping expand vm volume %s/%s", *vmVolume.ID, *vmVolume.Name), "currentSize", *vmVolume.Size, "expectedSize", *diskSize)
	}

	return true, nil
}

// resizeVMVolume sets the volume to the specified size.
func (r *ElfMachineReconciler) resizeVMVolume(ctx *context.MachineContext, vmVolume *models.VMVolume, diskSize int64, conditionType clusterv1.ConditionType) error {
	withTaskVMVolume, err := ctx.VMService.ResizeVMVolume(*vmVolume.ID, diskSize)
	if err != nil {
		conditions.MarkFalse(ctx.ElfMachine, conditionType, infrav1.ExpandingDiskFailedReason, clusterv1.ConditionSeverityWarning, err.Error())

		return errors.Wrapf(err, "failed to trigger expand size from %d to %d for vm volume %s/%s", *vmVolume.Size, diskSize, *vmVolume.ID, *vmVolume.Name)
	}

	conditions.MarkFalse(ctx.ElfMachine, conditionType, infrav1.ExpandingDiskReason, clusterv1.ConditionSeverityInfo, "")

	ctx.ElfMachine.SetTask(*withTaskVMVolume.TaskID)

	ctx.Logger.Info(fmt.Sprintf("Waiting for the vm volume %s/%s to be expanded", *vmVolume.ID, *vmVolume.Name), "taskRef", ctx.ElfMachine.Status.TaskRef, "oldSize", *vmVolume.Size, "newSize", diskSize)

	return nil
}

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
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"
	agentv1 "github.com/smartxworks/host-config-agent-api/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capiremote "sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/smartxworks/cluster-api-provider-elf/api/v1beta1"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/context"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/hostagent"
	"github.com/smartxworks/cluster-api-provider-elf/pkg/service"
	machineutil "github.com/smartxworks/cluster-api-provider-elf/pkg/util/machine"
)

func (r *ElfMachineReconciler) reconcileVMResources(ctx *context.MachineContext, vm *models.VM) (bool, error) {
	if ctx.ElfMachine.Status.Resources.Disk > 0 &&
		ctx.ElfMachine.Spec.DiskGiB > ctx.ElfMachine.Status.Resources.Disk {
		if !conditions.Has(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition) || conditions.IsTrue(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition) {
			conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.WaitingForResourcesHotUpdateReason, clusterv1.ConditionSeverityWarning, "")
		}
	}

	hotUpdatedCondition := conditions.Get(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition)
	if hotUpdatedCondition != nil &&
		hotUpdatedCondition.Reason == infrav1.WaitingForResourcesHotUpdateReason &&
		hotUpdatedCondition.Message != "" {
		ctx.Logger.V(2).Info("Waiting for hot updating resources", "message", hotUpdatedCondition.Message)

		return false, nil
	}

	if ok, err := r.reconcieVMNetworkDevices(ctx, vm); err != nil || !ok {
		return ok, err
	}

	if ok, err := r.reconcieVMVolume(ctx, vm, infrav1.ResourcesHotUpdatedCondition); err != nil || !ok {
		return ok, err
	}

	// Agent needs to wait for the node exists before it can run and execute commands.
	if machineutil.IsUpdatingElfMachineResources(ctx.ElfMachine) &&
		ctx.Machine.Status.NodeInfo == nil {
		ctx.Logger.Info("Waiting for node exists for host agent expand vm root partition")

		return false, nil
	}

	if ok, err := r.expandVMRootPartition(ctx); err != nil || !ok {
		return ok, err
	}

	if machineutil.IsUpdatingElfMachineResources(ctx.ElfMachine) {
		conditions.MarkTrue(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition)
	}

	return true, nil
}

func (r *ElfMachineReconciler) reconcieVMNetworkDevices(ctx *context.MachineContext, vm *models.VM) (bool, error) {
	vmNics, err := ctx.VMService.GetVMNics(*vm.ID)
	if err != nil {
		return false, err
	}

	devices := ctx.ElfMachine.Spec.Network.Devices
	if len(devices) > len(vmNics) {
		return false, r.addVMNetworkDevices(ctx, vm, vmNics)
	}

	if ok, err := r.setVMNetworkDeviceConfig(ctx, vmNics); err != nil || !ok {
		return ok, err
	}

	reason := conditions.GetReason(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition)
	if reason == "" {
		return true, nil
	} else if reason != infrav1.AddingVMNetworkDeviceReason &&
		reason != infrav1.AddingVMNetworkDeviceFailedReason &&
		reason != infrav1.SettingVMNetworkDeviceConfigFailedReason &&
		reason != infrav1.SettingVMNetworkDeviceConfigReason &&
		reason != infrav1.WaitingForNetworkAddressesReason {
		return true, nil
	}

	isWaitingForNetworkAddresses := false
	for i := 0; i < len(devices); i++ {
		if devices[i].NetworkType == infrav1.NetworkTypeNone {
			continue
		}

		if service.GetTowerString(vmNics[i].IPAddress) == "" {
			isWaitingForNetworkAddresses = true
			conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.WaitingForNetworkAddressesReason, clusterv1.ConditionSeverityInfo, "")
		}
	}

	var networkStatus []infrav1.NetworkStatus
	ipToMachineAddressMap := make(map[string]clusterv1.MachineAddress)
	for i := 0; i < len(vmNics); i++ {
		nic := vmNics[i]
		ip := service.GetTowerString(nic.IPAddress)

		// Add to Status.Network even if IP is empty.
		networkStatus = append(networkStatus, infrav1.NetworkStatus{
			IPAddrs: []string{ip},
			MACAddr: service.GetTowerString(nic.MacAddress),
		})

		if ip == "" {
			continue
		}

		ipToMachineAddressMap[ip] = clusterv1.MachineAddress{
			Type:    clusterv1.MachineInternalIP,
			Address: ip,
		}
	}
	ctx.ElfMachine.Status.Network = networkStatus

	addresses := make([]clusterv1.MachineAddress, 0, len(ipToMachineAddressMap))
	for _, machineAddress := range ipToMachineAddressMap {
		addresses = append(addresses, machineAddress)
	}
	ctx.ElfMachine.Status.Addresses = addresses

	if isWaitingForNetworkAddresses {
		ctx.Logger.V(1).Info("VM network is not ready yet", "nicStatus", ctx.ElfMachine.Status.Network)

		return false, nil
	}

	return true, nil
}

func (r *ElfMachineReconciler) addVMNetworkDevices(ctx *context.MachineContext, vm *models.VM, vmNics []*models.VMNic) error {
	reason := conditions.GetReason(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition)
	if reason == "" ||
		(reason != infrav1.AddingVMNetworkDeviceReason && reason != infrav1.AddingVMNetworkDeviceFailedReason) {
		conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.AddingVMNetworkDeviceReason, clusterv1.ConditionSeverityInfo, "")
		return nil
	}

	var newNics []*models.VMNicParams
	devices := ctx.ElfMachine.Spec.Network.Devices
	for i := len(vmNics); i < len(devices); i++ {
		device := devices[i]

		vlan, err := ctx.VMService.GetVlan(device.Vlan)
		if err != nil {
			return err
		}

		nic := &models.VMNicParams{
			Model:         models.NewVMNicModel(models.VMNicModelVIRTIO),
			Enabled:       service.TowerBool(true),
			Mirror:        service.TowerBool(false),
			ConnectVlanID: vlan.ID,
			MacAddress:    service.TowerString(device.MACAddr),
			SubnetMask:    service.TowerString(device.Netmask),
		}

		if len(device.IPAddrs) > 0 {
			nic.IPAddress = service.TowerString(device.IPAddrs[0])
		}

		newNics = append(newNics, nic)
	}

	withTaskVM, err := ctx.VMService.AddVMNics(*vm.ID, newNics)
	if err != nil {
		conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.AddingVMNetworkDeviceReason, clusterv1.ConditionSeverityWarning, err.Error())

		return errors.Wrapf(err, "failed to trigger add new nic to vm")
	}

	ctx.ElfMachine.SetTask(*withTaskVM.TaskID)

	ctx.Logger.Info("Waiting for the vm to be added new nic", "taskRef", ctx.ElfMachine.Status.TaskRef)

	return nil
}

func (r *ElfMachineReconciler) setVMNetworkDeviceConfig(ctx *context.MachineContext, vmNics []*models.VMNic) (bool, error) {
	reason := conditions.GetReason(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition)
	if reason == "" {
		return true, nil
	} else if reason != infrav1.AddingVMNetworkDeviceReason &&
		reason != infrav1.AddingVMNetworkDeviceFailedReason &&
		reason != infrav1.SettingVMNetworkDeviceConfigReason &&
		reason != infrav1.SettingVMNetworkDeviceConfigFailedReason {
		return true, nil
	}

	if reason != infrav1.SettingVMNetworkDeviceConfigFailedReason {
		conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.SettingVMNetworkDeviceConfigReason, clusterv1.ConditionSeverityInfo, "")
	}

	kubeClient, err := capiremote.NewClusterClient(ctx, "", ctx.Client, client.ObjectKey{Namespace: ctx.Cluster.Namespace, Name: ctx.Cluster.Name})
	if err != nil {
		return false, err
	}

	sort.Slice(vmNics, func(i, j int) bool {
		return *vmNics[i].Order <= *vmNics[j].Order
	})

	agentJob, err := hostagent.GetHostJob(ctx, kubeClient, ctx.ElfMachine.Namespace, hostagent.GetSetNetworkDeviceConfigJobName(ctx.ElfMachine, *vmNics[len(vmNics)-1].MacAddress))
	if err != nil && !apierrors.IsNotFound(err) {
		return false, err
	}

	if agentJob == nil {
		agentJob, err = hostagent.SetNetworkDeviceConfig(ctx, kubeClient, ctx.ElfMachine, *vmNics[len(vmNics)-1].MacAddress)
		if err != nil {
			conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.SettingVMNetworkDeviceConfigFailedReason, clusterv1.ConditionSeverityInfo, err.Error())

			return false, err
		}

		ctx.Logger.Info("Waiting for setting network device configuration", "hostAgentJob", agentJob.Name)

		return false, nil
	}

	switch agentJob.Status.Phase {
	case agentv1.PhaseSucceeded:
		ctx.Logger.Info("Set network device configuration succeeded", "hostAgentJob", agentJob.Name)
	case agentv1.PhaseFailed:
		conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.SettingVMNetworkDeviceConfigFailedReason, clusterv1.ConditionSeverityWarning, agentJob.Status.FailureMessage)
		ctx.Logger.Info("Set network device configuration failed, will try again after two minutes", "hostAgentJob", agentJob.Name, "failureMessage", agentJob.Status.FailureMessage)

		lastExecutionTime := agentJob.Status.LastExecutionTime
		if lastExecutionTime == nil {
			lastExecutionTime = &agentJob.CreationTimestamp
		}
		// Two minutes after the job fails, delete the job and try again.
		if time.Now().After(lastExecutionTime.Add(2 * time.Minute)) {
			if err := kubeClient.Delete(ctx, agentJob); err != nil {
				return false, errors.Wrapf(err, "failed to delete set network device configuration job %s/%s for retry", agentJob.Namespace, agentJob.Name)
			}
		}

		return false, nil
	default:
		ctx.Logger.Info("Waiting for setting network device configuration job done", "hostAgentJob", agentJob.Name, "jobStatus", agentJob.Status.Phase)

		return false, nil
	}

	return true, nil
}

// reconcieVMVolume ensures that the vm disk size is as expected.
//
// The conditionType param: VMProvisionedCondition/ResourcesHotUpdatedCondition.
func (r *ElfMachineReconciler) reconcieVMVolume(ctx *context.MachineContext, vm *models.VM, conditionType clusterv1.ConditionType) (bool, error) {
	vmDiskIDs := make([]string, len(vm.VMDisks))
	for i := 0; i < len(vm.VMDisks); i++ {
		vmDiskIDs[i] = *vm.VMDisks[i].ID
	}

	vmDisks, err := ctx.VMService.GetVMDisks(vmDiskIDs)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get disks for vm %s/%s", *vm.ID, *vm.Name)
	} else if len(vmDisks) == 0 {
		return false, errors.Errorf("no disks found for vm %s/%s", *vm.ID, *vm.Name)
	}

	vmVolume, err := ctx.VMService.GetVMVolume(*vmDisks[0].VMVolume.ID)
	if err != nil {
		return false, err
	}

	diskSize := service.ByteToGiB(*vmVolume.Size)
	ctx.ElfMachine.Status.Resources.Disk = diskSize

	if ctx.ElfMachine.Spec.DiskGiB < diskSize {
		ctx.Logger.V(3).Info(fmt.Sprintf("Current disk capacity is larger than expected, skipping expand vm volume %s/%s", *vmVolume.ID, *vmVolume.Name), "currentSize", diskSize, "expectedSize", ctx.ElfMachine.Spec.DiskGiB)
	} else if ctx.ElfMachine.Spec.DiskGiB > diskSize {
		return false, r.resizeVMVolume(ctx, vmVolume, *service.TowerDisk(ctx.ElfMachine.Spec.DiskGiB), conditionType)
	}

	return true, nil
}

// resizeVMVolume sets the volume to the specified size.
func (r *ElfMachineReconciler) resizeVMVolume(ctx *context.MachineContext, vmVolume *models.VMVolume, diskSize int64, conditionType clusterv1.ConditionType) error {
	reason := conditions.GetReason(ctx.ElfMachine, conditionType)
	if reason == "" ||
		(reason != infrav1.ExpandingVMDiskReason && reason != infrav1.ExpandingVMDiskFailedReason) {
		conditions.MarkFalse(ctx.ElfMachine, conditionType, infrav1.ExpandingVMDiskReason, clusterv1.ConditionSeverityInfo, "")

		// Save the conditionType first, and then expand the disk capacity.
		// This prevents the disk expansion from succeeding but failing to save the
		// conditionType, causing ElfMachine to not record the conditionType.
		return nil
	}

	if service.IsTowerResourcePerformingAnOperation(vmVolume.EntityAsyncStatus) {
		ctx.Logger.Info("Waiting for vm volume task done", "volume", fmt.Sprintf("%s/%s", *vmVolume.ID, *vmVolume.Name))

		return nil
	}

	withTaskVMVolume, err := ctx.VMService.ResizeVMVolume(*vmVolume.ID, diskSize)
	if err != nil {
		conditions.MarkFalse(ctx.ElfMachine, conditionType, infrav1.ExpandingVMDiskFailedReason, clusterv1.ConditionSeverityWarning, err.Error())

		return errors.Wrapf(err, "failed to trigger expand size from %d to %d for vm volume %s/%s", *vmVolume.Size, diskSize, *vmVolume.ID, *vmVolume.Name)
	}

	ctx.ElfMachine.SetTask(*withTaskVMVolume.TaskID)

	ctx.Logger.Info(fmt.Sprintf("Waiting for the vm volume %s/%s to be expanded", *vmVolume.ID, *vmVolume.Name), "taskRef", ctx.ElfMachine.Status.TaskRef, "oldSize", *vmVolume.Size, "newSize", diskSize)

	return nil
}

// expandVMRootPartition adds new disk capacity to root partition.
func (r *ElfMachineReconciler) expandVMRootPartition(ctx *context.MachineContext) (bool, error) {
	reason := conditions.GetReason(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition)
	if reason == "" {
		return true, nil
	} else if reason != infrav1.ExpandingVMDiskReason &&
		reason != infrav1.ExpandingVMDiskFailedReason &&
		reason != infrav1.ExpandingRootPartitionReason &&
		reason != infrav1.ExpandingRootPartitionFailedReason {
		return true, nil
	}

	if reason != infrav1.ExpandingRootPartitionFailedReason {
		conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.ExpandingRootPartitionReason, clusterv1.ConditionSeverityInfo, "")
	}

	kubeClient, err := capiremote.NewClusterClient(ctx, "", ctx.Client, client.ObjectKey{Namespace: ctx.Cluster.Namespace, Name: ctx.Cluster.Name})
	if err != nil {
		return false, err
	}

	agentJob, err := hostagent.GetHostJob(ctx, kubeClient, ctx.ElfMachine.Namespace, hostagent.GetExpandRootPartitionJobName(ctx.ElfMachine))
	if err != nil && !apierrors.IsNotFound(err) {
		return false, err
	}

	if agentJob == nil {
		agentJob, err = hostagent.ExpandRootPartition(ctx, kubeClient, ctx.ElfMachine)
		if err != nil {
			conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.ExpandingRootPartitionFailedReason, clusterv1.ConditionSeverityInfo, err.Error())

			return false, err
		}

		ctx.Logger.Info("Waiting for expanding root partition", "hostAgentJob", agentJob.Name)

		return false, nil
	}

	switch agentJob.Status.Phase {
	case agentv1.PhaseSucceeded:
		ctx.Logger.Info("Expand root partition to root succeeded", "hostAgentJob", agentJob.Name)
	case agentv1.PhaseFailed:
		conditions.MarkFalse(ctx.ElfMachine, infrav1.ResourcesHotUpdatedCondition, infrav1.ExpandingRootPartitionFailedReason, clusterv1.ConditionSeverityWarning, agentJob.Status.FailureMessage)
		ctx.Logger.Info("Expand root partition failed, will try again after two minutes", "hostAgentJob", agentJob.Name, "failureMessage", agentJob.Status.FailureMessage)

		lastExecutionTime := agentJob.Status.LastExecutionTime
		if lastExecutionTime == nil {
			lastExecutionTime = &agentJob.CreationTimestamp
		}
		// Two minutes after the job fails, delete the job and try again.
		if time.Now().After(lastExecutionTime.Add(2 * time.Minute)) {
			if err := kubeClient.Delete(ctx, agentJob); err != nil {
				return false, errors.Wrapf(err, "failed to delete expand root partition job %s/%s for retry", agentJob.Namespace, agentJob.Name)
			}
		}

		return false, nil
	default:
		ctx.Logger.Info("Waiting for expanding root partition job done", "hostAgentJob", agentJob.Name, "jobStatus", agentJob.Status.Phase)

		return false, nil
	}

	return true, nil
}

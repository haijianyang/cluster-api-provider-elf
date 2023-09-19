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
	goctx "context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/smartxworks/cluster-api-provider-elf/pkg/config"
)

const (
	vmCreationTimeout    = time.Minute * 6
	vmOperationRateLimit = time.Second * 6
	vmSilenceTime        = time.Minute * 5
	// When Tower gets a placement group name duplicate error, it means the ELF API is responding slow.
	// Tower will sync this placement group from ELF cluster immediately and the sync usually can complete within 1~2 minute.
	// So set the placement group creation retry interval to 5 minutes.
	placementGroupSilenceTime = time.Minute * 5
)

var vmTaskErrorCache = cache.New(5*time.Minute, 10*time.Minute)
var vmConcurrentCache = cache.New(5*time.Minute, 6*time.Minute)

var vmOperationLock sync.Mutex
var placementGroupOperationLock sync.Mutex

// acquireTicketForCreateVM returns whether virtual machine create operation
// can be performed.
func acquireTicketForCreateVM(vmName string, isControlPlaneVM bool) (bool, string) {
	vmOperationLock.Lock()
	defer vmOperationLock.Unlock()

	if _, found := vmTaskErrorCache.Get(getKeyForVMDuplicate(vmName)); found {
		return false, "Duplicate virtual machine detected"
	}

	// Only limit the concurrent of worker virtual machines.
	if isControlPlaneVM {
		return true, ""
	}

	concurrentCount := vmConcurrentCache.ItemCount()
	if concurrentCount >= config.MaxConcurrentVMCreations {
		return false, fmt.Sprintf("The number of concurrently created VMs has reached the limit %d", config.MaxConcurrentVMCreations)
	}

	vmConcurrentCache.Set(getKeyForVM(vmName), nil, vmCreationTimeout)

	return true, ""
}

// releaseTicketForCreateVM releases the virtual machine being created.
func releaseTicketForCreateVM(vmName string) {
	vmConcurrentCache.Delete(getKeyForVM(vmName))
}

// acquireTicketForUpdatingVM returns whether virtual machine update operation
// can be performed.
// Tower API currently does not have a concurrency limit for operations on the same virtual machine,
// which may cause task to fail.
func acquireTicketForUpdatingVM(vmName string) bool {
	if _, found := vmTaskErrorCache.Get(getKeyForVM(vmName)); found {
		return false
	}

	vmTaskErrorCache.Set(getKeyForVM(vmName), nil, vmOperationRateLimit)

	return true
}

// setVMDuplicate sets whether virtual machine is duplicated.
func setVMDuplicate(vmName string) {
	vmTaskErrorCache.Set(getKeyForVMDuplicate(vmName), nil, vmSilenceTime)
}

// acquireTicketForPlacementGroupOperation returns whether placement group operation
// can be performed.
func acquireTicketForPlacementGroupOperation(groupName string) bool {
	placementGroupOperationLock.Lock()
	defer placementGroupOperationLock.Unlock()

	if _, found := vmTaskErrorCache.Get(getKeyForPlacementGroup(groupName)); found {
		return false
	}

	vmTaskErrorCache.Set(getKeyForPlacementGroup(groupName), nil, cache.NoExpiration)

	return true
}

// releaseTicketForPlacementGroupOperation releases the placement group being operated.
func releaseTicketForPlacementGroupOperation(groupName string) {
	vmTaskErrorCache.Delete(getKeyForPlacementGroup(groupName))
}

// setPlacementGroupDuplicate sets whether placement group is duplicated.
func setPlacementGroupDuplicate(groupName string) {
	vmTaskErrorCache.Set(getKeyForPlacementGroupDuplicate(groupName), nil, placementGroupSilenceTime)
}

// canCreatePlacementGroup returns whether placement group creation can be performed.
func canCreatePlacementGroup(groupName string) bool {
	_, found := vmTaskErrorCache.Get(getKeyForPlacementGroupDuplicate(groupName))

	return !found
}

func getKeyForPlacementGroup(name string) string {
	return fmt.Sprintf("pg:%s", name)
}

func getKeyForPlacementGroupDuplicate(name string) string {
	return fmt.Sprintf("pg:duplicate:%s", name)
}

func getKeyForVM(name string) string {
	return fmt.Sprintf("vm:%s", name)
}

func getKeyForVMDuplicate(name string) string {
	return fmt.Sprintf("vm:duplicate:%s", name)
}

// acquireTicketForClusterOperation returns whether cluster operation
// can be performed.
func acquireTicketForClusterOperation(name string) bool {
	vmOperationLock.Lock()
	defer vmOperationLock.Unlock()

	if _, found := vmTaskErrorCache.Get(getKeyForCluster(name)); found {
		return false
	}

	vmTaskErrorCache.Set(getKeyForCluster(name), nil, cache.NoExpiration)

	return true
}

// releaseTicketForClusterpOperation releases the cluster being operated.
func releaseTicketForClusterpOperation(name string) {
	vmTaskErrorCache.Delete(getKeyForCluster(name))
}

func getKeyForCluster(name string) string {
	return fmt.Sprintf("cluster:%s", name)
}

const (
	freezionGPUConfigMapName = "freezion-gpus"
)

type freezionHostGPU struct {
	HostID       string    `json:"hostId"`
	GPUDeviceIDs []string  `json:"gpuDeviceIds"`
	FreezedAt    time.Time `json:"freezedAt"`
}

func freezeHostGPUDevices(ctx goctx.Context, c client.Client, clusterID, vmName, hostID string, gpuDeviceIDs []string) error {
	vmOperationLock.Lock()
	defer vmOperationLock.Unlock()

	freezionGPUConfigMap := &corev1.ConfigMap{}
	if err := c.Get(ctx, client.ObjectKey{
		Namespace: config.ProviderNamespace,
		Name:      freezionGPUConfigMapName,
	}, freezionGPUConfigMap); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		freezionGPUConfigMap.BinaryData = make(map[string][]byte)
	}

	freezionClusterGPUMap := make(map[string]freezionHostGPU)
	if _, ok := freezionGPUConfigMap.BinaryData[clusterID]; ok {
		if err := json.Unmarshal(freezionGPUConfigMap.BinaryData[clusterID], &freezionClusterGPUMap); err != nil {
			return err
		}
	}

	freezionClusterGPUMap[vmName] = freezionHostGPU{
		HostID:       hostID,
		GPUDeviceIDs: gpuDeviceIDs,
		FreezedAt:    time.Now(),
	}

	if bs, err := json.Marshal(freezionClusterGPUMap); err != nil {
		return err
	} else {
		freezionGPUConfigMap.BinaryData[clusterID] = bs
	}

	if freezionGPUConfigMap.CreationTimestamp.IsZero() {
		freezionGPUConfigMap.Namespace = config.ProviderNamespace
		freezionGPUConfigMap.Name = freezionGPUConfigMapName

		return c.Create(ctx, freezionGPUConfigMap)
	}

	return c.Patch(ctx, freezionGPUConfigMap, client.Merge)
}

func unfreezeHostGPUDevices(ctx goctx.Context, c client.Client, clusterID, vmName string) error {
	vmOperationLock.Lock()
	defer vmOperationLock.Unlock()

	freezionGPUConfigMap := &corev1.ConfigMap{}
	if err := c.Get(ctx, client.ObjectKey{
		Namespace: config.ProviderNamespace,
		Name:      freezionGPUConfigMapName,
	}, freezionGPUConfigMap); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	if _, ok := freezionGPUConfigMap.BinaryData[clusterID]; !ok {
		return nil
	}

	freezionClusterGPUMap := make(map[string]freezionHostGPU)
	if _, ok := freezionGPUConfigMap.BinaryData[clusterID]; ok {
		if err := json.Unmarshal(freezionGPUConfigMap.BinaryData[clusterID], &freezionClusterGPUMap); err != nil {
			return err
		}
	}

	delete(freezionClusterGPUMap, vmName)

	if bs, err := json.Marshal(freezionClusterGPUMap); err != nil {
		return err
	} else {
		freezionGPUConfigMap.BinaryData[clusterID] = bs
	}

	return c.Patch(ctx, freezionGPUConfigMap, client.Merge)
}

func getFreezeClusterGPUDevices(ctx goctx.Context, c client.Client, clusterID string) (sets.Set[string], error) {
	vmOperationLock.Lock()
	defer vmOperationLock.Unlock()

	gpuIDs := sets.Set[string]{}

	freezionGPUConfigMap := &corev1.ConfigMap{}
	if err := c.Get(ctx, client.ObjectKey{
		Namespace: config.ProviderNamespace,
		Name:      freezionGPUConfigMapName,
	}, freezionGPUConfigMap); err != nil {
		if apierrors.IsNotFound(err) {
			return gpuIDs, nil
		}

		return nil, err
	}

	freezionClusterGPUMap := make(map[string]freezionHostGPU)
	if _, ok := freezionGPUConfigMap.BinaryData[clusterID]; ok {
		if err := json.Unmarshal(freezionGPUConfigMap.BinaryData[clusterID], &freezionClusterGPUMap); err != nil {
			return nil, err
		}
	}

	hasStaleGPU := false
	for vmName, freezionHostGPU := range freezionClusterGPUMap {
		if time.Now().Before(freezionHostGPU.FreezedAt.Add(vmCreationTimeout)) {
			gpuIDs.Insert(freezionHostGPU.GPUDeviceIDs...)
		} else {
			delete(freezionClusterGPUMap, vmName)
			hasStaleGPU = true
		}
	}

	if hasStaleGPU {
		if bs, err := json.Marshal(freezionClusterGPUMap); err != nil {
			return nil, err
		} else {
			freezionGPUConfigMap.BinaryData[clusterID] = bs
		}

		if err := c.Patch(ctx, freezionGPUConfigMap, client.Merge); err != nil {
			return nil, err
		}
	}

	return gpuIDs, nil
}

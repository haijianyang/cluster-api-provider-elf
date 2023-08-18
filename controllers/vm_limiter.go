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
	"fmt"
	"sync"
	"time"

	"github.com/smartxworks/cluster-api-provider-elf/pkg/config"
)

const (
	creationTimeout      = time.Minute * 6
	vmOperationRateLimit = time.Second * 6
	vmSilenceTime        = time.Minute * 5
	// When Tower gets a placement group name duplicate error, it means the ELF API is responding slow.
	// Tower will sync this placement group from ELF cluster immediately and the sync usually can complete within 1~2 minute.
	// So set the placement group creation retry interval to 5 minutes.
	placementGroupSilenceTime = time.Minute * 5
)

var vmConcurrentMap = make(map[string]time.Time)
var vmOperationMap = make(map[string]time.Time)
var vmOperationLock sync.Mutex

var placementGroupOperationMap = make(map[string]time.Time)
var placementGroupOperationLock sync.Mutex

// acquireTicketForCreateVM returns whether virtual machine create operation
// can be performed.
func acquireTicketForCreateVM(vmName string, isControlPlaneVM bool) (bool, string) {
	vmOperationLock.Lock()
	defer vmOperationLock.Unlock()

	if timestamp, ok := vmOperationMap[getCreationLockKey(vmName)]; ok {
		if !time.Now().After(timestamp.Add(vmSilenceTime)) {
			return false, "Duplicate virtual machine detected"
		}
		delete(vmOperationMap, getCreationLockKey(vmName))
	}

	// Only limit the concurrent of worker virtual machines.
	if isControlPlaneVM {
		return true, ""
	}

	if len(vmConcurrentMap) >= config.MaxConcurrentVMCreations {
		for name, timestamp := range vmConcurrentMap {
			if !time.Now().Before(timestamp.Add(creationTimeout)) {
				delete(vmConcurrentMap, name)
			}
		}
	}

	if len(vmConcurrentMap) >= config.MaxConcurrentVMCreations {
		return false, "The number of concurrently created VMs has reached the limit"
	}

	vmConcurrentMap[vmName] = time.Now()

	return true, ""
}

// releaseTicketForCreateVM releases the virtual machine being created.
func releaseTicketForCreateVM(vmName string) {
	vmOperationLock.Lock()
	defer vmOperationLock.Unlock()

	delete(vmConcurrentMap, vmName)
}

// acquireTicketForUpdatingVM returns whether virtual machine update operation
// can be performed.
// Tower API currently does not have a concurrency limit for operations on the same virtual machine,
// which may cause task to fail.
func acquireTicketForUpdatingVM(vmName string) bool {
	vmOperationLock.Lock()
	defer vmOperationLock.Unlock()

	if timestamp, ok := vmOperationMap[vmName]; ok {
		if !time.Now().After(timestamp.Add(vmOperationRateLimit)) {
			return false
		}
	}

	vmOperationMap[vmName] = time.Now()

	return true
}

// setVMDuplicate sets whether virtual machine is duplicated.
func setVMDuplicate(vmName string) {
	vmOperationLock.Lock()
	defer vmOperationLock.Unlock()

	vmOperationMap[getCreationLockKey(vmName)] = time.Now()
}

// acquireTicketForPlacementGroupOperation returns whether placement group operation
// can be performed.
func acquireTicketForPlacementGroupOperation(groupName string) bool {
	placementGroupOperationLock.Lock()
	defer placementGroupOperationLock.Unlock()

	if _, ok := placementGroupOperationMap[groupName]; ok {
		return false
	}

	placementGroupOperationMap[groupName] = time.Now()

	return true
}

// releaseTicketForPlacementGroupOperation releases the placement group being operated.
func releaseTicketForPlacementGroupOperation(groupName string) {
	placementGroupOperationLock.Lock()
	defer placementGroupOperationLock.Unlock()

	delete(placementGroupOperationMap, groupName)
}

// setPlacementGroupDuplicate sets whether placement group is duplicated.
func setPlacementGroupDuplicate(groupName string) {
	placementGroupOperationLock.Lock()
	defer placementGroupOperationLock.Unlock()

	placementGroupOperationMap[getCreationLockKey(groupName)] = time.Now()
}

// canCreatePlacementGroup returns whether placement group creation can be performed.
func canCreatePlacementGroup(groupName string) bool {
	placementGroupOperationLock.Lock()
	defer placementGroupOperationLock.Unlock()

	key := getCreationLockKey(groupName)
	if timestamp, ok := placementGroupOperationMap[key]; ok {
		if time.Now().Before(timestamp.Add(placementGroupSilenceTime)) {
			return false
		}

		delete(placementGroupOperationMap, key)
	}

	return true
}

func getCreationLockKey(name string) string {
	return fmt.Sprintf("%s:creation", name)
}

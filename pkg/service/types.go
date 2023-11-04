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

package service

type GPUDeviceVM struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	AllocatedCount int32  `json:"allocatedCount"`
}

type GPUDeviceInfo struct {
	ID           string `json:"id"`
	HostID       string `json:"hostId"`
	Model        string `json:"model"`
	VGPUTypeName string `json:"vGpuTypeName"`
	// AllocatedCount  the number that has been allocated.
	// For GPU devices, can be 0 or 1.
	// For vGPU devices, can larger than vgpuInstanceNum.
	AllocatedCount int32 `json:"allocatedCount"`
	// AvailableCount is the number of GPU that can be allocated.
	// For GPU devices, can be 0 or 1.
	// For vGPU devices, can be 0 - vgpuInstanceNum.
	AvailableCount int32 `json:"availableCount"`
	// VMs(including STOPPED) allocated to the current GPU.
	VMs []GPUDeviceVM `json:"vms"`
}

func (g *GPUDeviceInfo) HasNoVMs() bool {
	return len(g.VMs) == 0
}

func (g *GPUDeviceInfo) ContainsVM(vm string) bool {
	for i := 0; i < len(g.VMs); i++ {
		if g.VMs[i].ID == vm || g.VMs[i].Name == vm {
			return true
		}
	}

	return false
}

// GPUDeviceInfos is a set of GPUDeviceInfos.
type GPUDeviceInfos map[string]*GPUDeviceInfo

// NewGPUDeviceInfos creates a GPUDeviceInfos. from a list of values.
func NewGPUDeviceInfos(gpuDeviceInfo ...*GPUDeviceInfo) GPUDeviceInfos {
	ss := make(GPUDeviceInfos, len(gpuDeviceInfo))
	ss.Insert(gpuDeviceInfo...)
	return ss
}

func (s GPUDeviceInfos) Insert(gpuDeviceInfos ...*GPUDeviceInfo) {
	for i := range gpuDeviceInfos {
		if gpuDeviceInfos[i] != nil {
			g := gpuDeviceInfos[i]
			s[g.ID] = g
		}
	}
}

// UnsortedList returns the slice with contents in random order.
func (s GPUDeviceInfos) UnsortedList() []*GPUDeviceInfo {
	res := make([]*GPUDeviceInfo, 0, len(s))
	for _, value := range s {
		res = append(res, value)
	}
	return res
}

// Get returns a GPUDeviceInfo of the specified gpuID.
func (s GPUDeviceInfos) Get(gpuID string) *GPUDeviceInfo {
	if gpuDeviceInfo, ok := s[gpuID]; ok {
		return gpuDeviceInfo
	}
	return nil
}

func (s GPUDeviceInfos) Contains(gpuID string) bool {
	_, ok := s[gpuID]
	return ok
}

func (s GPUDeviceInfos) Len() int {
	return len(s)
}

func (s GPUDeviceInfos) Iterate(fn func(*GPUDeviceInfo)) {
	for _, g := range s {
		fn(g)
	}
}

// Filter returns a GPUDeviceInfos containing only the GPUDeviceInfos that match all of the given GPUDeviceInfoFilters.
func (s GPUDeviceInfos) Filter(filters ...GPUDeviceInfoFilterFunc) GPUDeviceInfos {
	return newFilteredGPUDeviceInfoCollection(GPUDeviceInfoFilterAnd(filters...), s.UnsortedList()...)
}

// newFilteredGPUDeviceInfoCollection creates a GPUDeviceInfos from a filtered list of values.
func newFilteredGPUDeviceInfoCollection(filter GPUDeviceInfoFilterFunc, gpuDeviceInfos ...*GPUDeviceInfo) GPUDeviceInfos {
	ss := make(GPUDeviceInfos, len(gpuDeviceInfos))
	for i := range gpuDeviceInfos {
		g := gpuDeviceInfos[i]
		if filter(g) {
			ss.Insert(g)
		}
	}
	return ss
}

// GPUDeviceInfoFilterFunc is the functon definition for a filter.
type GPUDeviceInfoFilterFunc func(*GPUDeviceInfo) bool

// GPUDeviceInfoFilterAnd returns a filter that returns true if all of the given filters returns true.
func GPUDeviceInfoFilterAnd(filters ...GPUDeviceInfoFilterFunc) GPUDeviceInfoFilterFunc {
	return func(g *GPUDeviceInfo) bool {
		for _, f := range filters {
			if !f(g) {
				return false
			}
		}
		return true
	}
}

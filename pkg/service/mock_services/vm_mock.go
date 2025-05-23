// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/smartxworks/cluster-api-provider-elf/pkg/service/vm.go

// Package mock_services is a generated GoMock package.
package mock_services

import (
	context "context"
	reflect "reflect"
	time "time"

	gomock "github.com/golang/mock/gomock"
	models "github.com/smartxworks/cloudtower-go-sdk/v2/models"
	v1beta1 "github.com/smartxworks/cluster-api-provider-elf/api/v1beta1"
	service "github.com/smartxworks/cluster-api-provider-elf/pkg/service"
)

// MockVMService is a mock of VMService interface.
type MockVMService struct {
	ctrl     *gomock.Controller
	recorder *MockVMServiceMockRecorder
}

// MockVMServiceMockRecorder is the mock recorder for MockVMService.
type MockVMServiceMockRecorder struct {
	mock *MockVMService
}

// NewMockVMService creates a new mock instance.
func NewMockVMService(ctrl *gomock.Controller) *MockVMService {
	mock := &MockVMService{ctrl: ctrl}
	mock.recorder = &MockVMServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockVMService) EXPECT() *MockVMServiceMockRecorder {
	return m.recorder
}

// AddGPUDevices mocks base method.
func (m *MockVMService) AddGPUDevices(id string, gpuDeviceInfo []*service.GPUDeviceInfo) (*models.Task, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddGPUDevices", id, gpuDeviceInfo)
	ret0, _ := ret[0].(*models.Task)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddGPUDevices indicates an expected call of AddGPUDevices.
func (mr *MockVMServiceMockRecorder) AddGPUDevices(id, gpuDeviceInfo interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddGPUDevices", reflect.TypeOf((*MockVMService)(nil).AddGPUDevices), id, gpuDeviceInfo)
}

// AddLabelsToVM mocks base method.
func (m *MockVMService) AddLabelsToVM(vmID string, labels []string) (*models.Task, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddLabelsToVM", vmID, labels)
	ret0, _ := ret[0].(*models.Task)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddLabelsToVM indicates an expected call of AddLabelsToVM.
func (mr *MockVMServiceMockRecorder) AddLabelsToVM(vmID, labels interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddLabelsToVM", reflect.TypeOf((*MockVMService)(nil).AddLabelsToVM), vmID, labels)
}

// AddVMsToPlacementGroup mocks base method.
func (m *MockVMService) AddVMsToPlacementGroup(placementGroup *models.VMPlacementGroup, vmIDs []string) (*models.Task, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddVMsToPlacementGroup", placementGroup, vmIDs)
	ret0, _ := ret[0].(*models.Task)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddVMsToPlacementGroup indicates an expected call of AddVMsToPlacementGroup.
func (mr *MockVMServiceMockRecorder) AddVMsToPlacementGroup(placementGroup, vmIDs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddVMsToPlacementGroup", reflect.TypeOf((*MockVMService)(nil).AddVMsToPlacementGroup), placementGroup, vmIDs)
}

// CleanUnusedLabels mocks base method.
func (m *MockVMService) CleanUnusedLabels(keys []string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CleanUnusedLabels", keys)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CleanUnusedLabels indicates an expected call of CleanUnusedLabels.
func (mr *MockVMServiceMockRecorder) CleanUnusedLabels(keys interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CleanUnusedLabels", reflect.TypeOf((*MockVMService)(nil).CleanUnusedLabels), keys)
}

// Clone mocks base method.
func (m *MockVMService) Clone(elfCluster *v1beta1.ElfCluster, elfMachine *v1beta1.ElfMachine, bootstrapData, host string, machineGPUDevices []*service.GPUDeviceInfo) (*models.WithTaskVM, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Clone", elfCluster, elfMachine, bootstrapData, host, machineGPUDevices)
	ret0, _ := ret[0].(*models.WithTaskVM)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Clone indicates an expected call of Clone.
func (mr *MockVMServiceMockRecorder) Clone(elfCluster, elfMachine, bootstrapData, host, machineGPUDevices interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Clone", reflect.TypeOf((*MockVMService)(nil).Clone), elfCluster, elfMachine, bootstrapData, host, machineGPUDevices)
}

// CreateVMPlacementGroup mocks base method.
func (m *MockVMService) CreateVMPlacementGroup(name, clusterID string, vmPolicy models.VMVMPolicy) (*models.WithTaskVMPlacementGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateVMPlacementGroup", name, clusterID, vmPolicy)
	ret0, _ := ret[0].(*models.WithTaskVMPlacementGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateVMPlacementGroup indicates an expected call of CreateVMPlacementGroup.
func (mr *MockVMServiceMockRecorder) CreateVMPlacementGroup(name, clusterID, vmPolicy interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateVMPlacementGroup", reflect.TypeOf((*MockVMService)(nil).CreateVMPlacementGroup), name, clusterID, vmPolicy)
}

// Delete mocks base method.
func (m *MockVMService) Delete(uuid string) (*models.Task, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", uuid)
	ret0, _ := ret[0].(*models.Task)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Delete indicates an expected call of Delete.
func (mr *MockVMServiceMockRecorder) Delete(uuid interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockVMService)(nil).Delete), uuid)
}

// DeleteLabel mocks base method.
func (m *MockVMService) DeleteLabel(key, value string, strict bool) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteLabel", key, value, strict)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteLabel indicates an expected call of DeleteLabel.
func (mr *MockVMServiceMockRecorder) DeleteLabel(key, value, strict interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteLabel", reflect.TypeOf((*MockVMService)(nil).DeleteLabel), key, value, strict)
}

// DeleteVMPlacementGroupByID mocks base method.
func (m *MockVMService) DeleteVMPlacementGroupByID(ctx context.Context, id string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteVMPlacementGroupByID", ctx, id)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteVMPlacementGroupByID indicates an expected call of DeleteVMPlacementGroupByID.
func (mr *MockVMServiceMockRecorder) DeleteVMPlacementGroupByID(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteVMPlacementGroupByID", reflect.TypeOf((*MockVMService)(nil).DeleteVMPlacementGroupByID), ctx, id)
}

// DeleteVMPlacementGroupsByNamePrefix mocks base method.
func (m *MockVMService) DeleteVMPlacementGroupsByNamePrefix(ctx context.Context, placementGroupName string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteVMPlacementGroupsByNamePrefix", ctx, placementGroupName)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteVMPlacementGroupsByNamePrefix indicates an expected call of DeleteVMPlacementGroupsByNamePrefix.
func (mr *MockVMServiceMockRecorder) DeleteVMPlacementGroupsByNamePrefix(ctx, placementGroupName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteVMPlacementGroupsByNamePrefix", reflect.TypeOf((*MockVMService)(nil).DeleteVMPlacementGroupsByNamePrefix), ctx, placementGroupName)
}

// FindByIDs mocks base method.
func (m *MockVMService) FindByIDs(ids []string) ([]*models.VM, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByIDs", ids)
	ret0, _ := ret[0].([]*models.VM)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindByIDs indicates an expected call of FindByIDs.
func (mr *MockVMServiceMockRecorder) FindByIDs(ids interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByIDs", reflect.TypeOf((*MockVMService)(nil).FindByIDs), ids)
}

// FindVMsByName mocks base method.
func (m *MockVMService) FindVMsByName(name string) ([]*models.VM, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindVMsByName", name)
	ret0, _ := ret[0].([]*models.VM)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindVMsByName indicates an expected call of FindVMsByName.
func (mr *MockVMServiceMockRecorder) FindVMsByName(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindVMsByName", reflect.TypeOf((*MockVMService)(nil).FindVMsByName), name)
}

// Get mocks base method.
func (m *MockVMService) Get(id string) (*models.VM, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", id)
	ret0, _ := ret[0].(*models.VM)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockVMServiceMockRecorder) Get(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockVMService)(nil).Get), id)
}

// GetByName mocks base method.
func (m *MockVMService) GetByName(name string) (*models.VM, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByName", name)
	ret0, _ := ret[0].(*models.VM)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByName indicates an expected call of GetByName.
func (mr *MockVMServiceMockRecorder) GetByName(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByName", reflect.TypeOf((*MockVMService)(nil).GetByName), name)
}

// GetCluster mocks base method.
func (m *MockVMService) GetCluster(id string) (*models.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCluster", id)
	ret0, _ := ret[0].(*models.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCluster indicates an expected call of GetCluster.
func (mr *MockVMServiceMockRecorder) GetCluster(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCluster", reflect.TypeOf((*MockVMService)(nil).GetCluster), id)
}

// GetClusterZones mocks base method.
func (m *MockVMService) GetClusterZones(id string) ([]*models.Zone, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetClusterZones", id)
	ret0, _ := ret[0].([]*models.Zone)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetClusterZones indicates an expected call of GetClusterZones.
func (mr *MockVMServiceMockRecorder) GetClusterZones(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetClusterZones", reflect.TypeOf((*MockVMService)(nil).GetClusterZones), id)
}

// GetGPUDevicesAllocationInfoByHostIDs mocks base method.
func (m *MockVMService) GetGPUDevicesAllocationInfoByHostIDs(hostIDs []string, gpuDeviceUsage models.GpuDeviceUsage) (service.GPUVMInfos, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGPUDevicesAllocationInfoByHostIDs", hostIDs, gpuDeviceUsage)
	ret0, _ := ret[0].(service.GPUVMInfos)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetGPUDevicesAllocationInfoByHostIDs indicates an expected call of GetGPUDevicesAllocationInfoByHostIDs.
func (mr *MockVMServiceMockRecorder) GetGPUDevicesAllocationInfoByHostIDs(hostIDs, gpuDeviceUsage interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGPUDevicesAllocationInfoByHostIDs", reflect.TypeOf((*MockVMService)(nil).GetGPUDevicesAllocationInfoByHostIDs), hostIDs, gpuDeviceUsage)
}

// GetGPUDevicesAllocationInfoByIDs mocks base method.
func (m *MockVMService) GetGPUDevicesAllocationInfoByIDs(gpuIDs []string) (service.GPUVMInfos, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGPUDevicesAllocationInfoByIDs", gpuIDs)
	ret0, _ := ret[0].(service.GPUVMInfos)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetGPUDevicesAllocationInfoByIDs indicates an expected call of GetGPUDevicesAllocationInfoByIDs.
func (mr *MockVMServiceMockRecorder) GetGPUDevicesAllocationInfoByIDs(gpuIDs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGPUDevicesAllocationInfoByIDs", reflect.TypeOf((*MockVMService)(nil).GetGPUDevicesAllocationInfoByIDs), gpuIDs)
}

// GetHost mocks base method.
func (m *MockVMService) GetHost(id string) (*models.Host, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHost", id)
	ret0, _ := ret[0].(*models.Host)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHost indicates an expected call of GetHost.
func (mr *MockVMServiceMockRecorder) GetHost(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHost", reflect.TypeOf((*MockVMService)(nil).GetHost), id)
}

// GetHostsByCluster mocks base method.
func (m *MockVMService) GetHostsByCluster(clusterID string) (service.Hosts, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHostsByCluster", clusterID)
	ret0, _ := ret[0].(service.Hosts)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHostsByCluster indicates an expected call of GetHostsByCluster.
func (mr *MockVMServiceMockRecorder) GetHostsByCluster(clusterID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHostsByCluster", reflect.TypeOf((*MockVMService)(nil).GetHostsByCluster), clusterID)
}

// GetTask mocks base method.
func (m *MockVMService) GetTask(id string) (*models.Task, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTask", id)
	ret0, _ := ret[0].(*models.Task)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTask indicates an expected call of GetTask.
func (mr *MockVMServiceMockRecorder) GetTask(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTask", reflect.TypeOf((*MockVMService)(nil).GetTask), id)
}

// GetVMDisks mocks base method.
func (m *MockVMService) GetVMDisks(vmDiskIDs []string) ([]*models.VMDisk, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVMDisks", vmDiskIDs)
	ret0, _ := ret[0].([]*models.VMDisk)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVMDisks indicates an expected call of GetVMDisks.
func (mr *MockVMServiceMockRecorder) GetVMDisks(vmDiskIDs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVMDisks", reflect.TypeOf((*MockVMService)(nil).GetVMDisks), vmDiskIDs)
}

// GetVMGPUAllocationInfo mocks base method.
func (m *MockVMService) GetVMGPUAllocationInfo(id string) (*models.VMGpuInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVMGPUAllocationInfo", id)
	ret0, _ := ret[0].(*models.VMGpuInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVMGPUAllocationInfo indicates an expected call of GetVMGPUAllocationInfo.
func (mr *MockVMServiceMockRecorder) GetVMGPUAllocationInfo(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVMGPUAllocationInfo", reflect.TypeOf((*MockVMService)(nil).GetVMGPUAllocationInfo), id)
}

// GetVMNics mocks base method.
func (m *MockVMService) GetVMNics(vmID string) ([]*models.VMNic, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVMNics", vmID)
	ret0, _ := ret[0].([]*models.VMNic)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVMNics indicates an expected call of GetVMNics.
func (mr *MockVMServiceMockRecorder) GetVMNics(vmID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVMNics", reflect.TypeOf((*MockVMService)(nil).GetVMNics), vmID)
}

// GetVMPlacementGroup mocks base method.
func (m *MockVMService) GetVMPlacementGroup(name string) (*models.VMPlacementGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVMPlacementGroup", name)
	ret0, _ := ret[0].(*models.VMPlacementGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVMPlacementGroup indicates an expected call of GetVMPlacementGroup.
func (mr *MockVMServiceMockRecorder) GetVMPlacementGroup(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVMPlacementGroup", reflect.TypeOf((*MockVMService)(nil).GetVMPlacementGroup), name)
}

// GetVMTemplate mocks base method.
func (m *MockVMService) GetVMTemplate(template string) (*models.ContentLibraryVMTemplate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVMTemplate", template)
	ret0, _ := ret[0].(*models.ContentLibraryVMTemplate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVMTemplate indicates an expected call of GetVMTemplate.
func (mr *MockVMServiceMockRecorder) GetVMTemplate(template interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVMTemplate", reflect.TypeOf((*MockVMService)(nil).GetVMTemplate), template)
}

// GetVMVolume mocks base method.
func (m *MockVMService) GetVMVolume(vmVolumeID string) (*models.VMVolume, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVMVolume", vmVolumeID)
	ret0, _ := ret[0].(*models.VMVolume)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVMVolume indicates an expected call of GetVMVolume.
func (mr *MockVMServiceMockRecorder) GetVMVolume(vmVolumeID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVMVolume", reflect.TypeOf((*MockVMService)(nil).GetVMVolume), vmVolumeID)
}

// GetVlan mocks base method.
func (m *MockVMService) GetVlan(id string) (*models.Vlan, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVlan", id)
	ret0, _ := ret[0].(*models.Vlan)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVlan indicates an expected call of GetVlan.
func (mr *MockVMServiceMockRecorder) GetVlan(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVlan", reflect.TypeOf((*MockVMService)(nil).GetVlan), id)
}

// Migrate mocks base method.
func (m *MockVMService) Migrate(vmID, hostID string) (*models.WithTaskVM, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Migrate", vmID, hostID)
	ret0, _ := ret[0].(*models.WithTaskVM)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Migrate indicates an expected call of Migrate.
func (mr *MockVMServiceMockRecorder) Migrate(vmID, hostID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Migrate", reflect.TypeOf((*MockVMService)(nil).Migrate), vmID, hostID)
}

// PowerOff mocks base method.
func (m *MockVMService) PowerOff(uuid string) (*models.Task, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PowerOff", uuid)
	ret0, _ := ret[0].(*models.Task)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PowerOff indicates an expected call of PowerOff.
func (mr *MockVMServiceMockRecorder) PowerOff(uuid interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PowerOff", reflect.TypeOf((*MockVMService)(nil).PowerOff), uuid)
}

// PowerOn mocks base method.
func (m *MockVMService) PowerOn(id, hostID string) (*models.Task, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PowerOn", id, hostID)
	ret0, _ := ret[0].(*models.Task)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PowerOn indicates an expected call of PowerOn.
func (mr *MockVMServiceMockRecorder) PowerOn(id, hostID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PowerOn", reflect.TypeOf((*MockVMService)(nil).PowerOn), id, hostID)
}

// RemoveGPUDevices mocks base method.
func (m *MockVMService) RemoveGPUDevices(id string, gpus []*models.VMGpuOperationParams) (*models.Task, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveGPUDevices", id, gpus)
	ret0, _ := ret[0].(*models.Task)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RemoveGPUDevices indicates an expected call of RemoveGPUDevices.
func (mr *MockVMServiceMockRecorder) RemoveGPUDevices(id, gpus interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveGPUDevices", reflect.TypeOf((*MockVMService)(nil).RemoveGPUDevices), id, gpus)
}

// ResizeVMVolume mocks base method.
func (m *MockVMService) ResizeVMVolume(vmVolumeID string, size int64) (*models.WithTaskVMVolume, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResizeVMVolume", vmVolumeID, size)
	ret0, _ := ret[0].(*models.WithTaskVMVolume)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ResizeVMVolume indicates an expected call of ResizeVMVolume.
func (mr *MockVMServiceMockRecorder) ResizeVMVolume(vmVolumeID, size interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResizeVMVolume", reflect.TypeOf((*MockVMService)(nil).ResizeVMVolume), vmVolumeID, size)
}

// ShutDown mocks base method.
func (m *MockVMService) ShutDown(uuid string) (*models.Task, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ShutDown", uuid)
	ret0, _ := ret[0].(*models.Task)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ShutDown indicates an expected call of ShutDown.
func (mr *MockVMServiceMockRecorder) ShutDown(uuid interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ShutDown", reflect.TypeOf((*MockVMService)(nil).ShutDown), uuid)
}

// UpdateVM mocks base method.
func (m *MockVMService) UpdateVM(vm *models.VM, elfMachine *v1beta1.ElfMachine) (*models.WithTaskVM, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateVM", vm, elfMachine)
	ret0, _ := ret[0].(*models.WithTaskVM)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateVM indicates an expected call of UpdateVM.
func (mr *MockVMServiceMockRecorder) UpdateVM(vm, elfMachine interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateVM", reflect.TypeOf((*MockVMService)(nil).UpdateVM), vm, elfMachine)
}

// UpsertLabel mocks base method.
func (m *MockVMService) UpsertLabel(key, value string) (*models.Label, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpsertLabel", key, value)
	ret0, _ := ret[0].(*models.Label)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpsertLabel indicates an expected call of UpsertLabel.
func (mr *MockVMServiceMockRecorder) UpsertLabel(key, value interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpsertLabel", reflect.TypeOf((*MockVMService)(nil).UpsertLabel), key, value)
}

// WaitTask mocks base method.
func (m *MockVMService) WaitTask(ctx context.Context, id string, timeout, interval time.Duration) (*models.Task, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitTask", ctx, id, timeout, interval)
	ret0, _ := ret[0].(*models.Task)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WaitTask indicates an expected call of WaitTask.
func (mr *MockVMServiceMockRecorder) WaitTask(ctx, id, timeout, interval interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitTask", reflect.TypeOf((*MockVMService)(nil).WaitTask), ctx, id, timeout, interval)
}

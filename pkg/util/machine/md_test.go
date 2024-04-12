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

package machine

import (
	goctx "context"
	"testing"

	"github.com/onsi/gomega"

	"github.com/smartxworks/cluster-api-provider-elf/test/fake"
)

func TestGetMDByMachine(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	ctx := goctx.TODO()
	elfCluster, cluster := fake.NewClusterObjects()
	_, machine := fake.NewMachineObjects(elfCluster, cluster)
	machineDeployment := fake.NewMD()
	fake.ToWorkerMachine(machine, machineDeployment)
	ctrlMgrCtx := fake.NewControllerManagerContext(machineDeployment)

	t.Run("should return md", func(t *testing.T) {
		md, err := GetMDByMachine(ctx, ctrlMgrCtx.Client, machine)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(md.Name).To(gomega.Equal(machineDeployment.Name))
	})
}

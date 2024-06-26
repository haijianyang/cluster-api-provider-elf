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

package context

import (
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ControllerManagerContext is the context of the controller that owns the
// controllers.
type ControllerManagerContext struct {
	// Namespace is the namespace in which the resource is located responsible
	// for running the controller manager.
	Namespace string

	// Name is the name of the controller manager.
	Name string

	// LeaderElectionID is the information used to identify the object
	// responsible for synchronizing leader election.
	LeaderElectionID string

	// LeaderElectionNamespace is the namespace in which the LeaderElection
	// object is located.
	LeaderElectionNamespace string

	// WatchNamespaces are the namespaces the controllers watches for changes. If
	// no value is specified then all namespaces are watched.
	WatchNamespaces map[string]cache.Config

	// Client is the controller manager's client.
	Client client.Client

	// Scheme is the controller manager's API scheme.
	Scheme *runtime.Scheme

	// WatchFilterValue is used to filter incoming objects by label.
	WatchFilterValue string
}

// String returns ControllerManagerName.
func (r *ControllerManagerContext) String() string {
	return r.Name
}

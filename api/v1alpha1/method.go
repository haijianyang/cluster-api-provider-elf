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

package v1alpha1

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	valuesYamlKey = "values.yaml"
)

func (in *YAMLText) IsEmpty() bool {
	if in == nil {
		return true
	}
	if in.SecretRef != nil && in.SecretRef.Name == "" && in.Content == "" {
		return true
	}
	return in.SecretRef == nil && in.Content == ""
}

func (in *YAMLText) getAddonSecretNamespacedName(defaultNamespace string) (apitypes.NamespacedName, bool) {
	if in.SecretRef != nil && in.SecretRef.Name != "" {
		result := apitypes.NamespacedName{
			Namespace: in.SecretRef.Namespace,
			Name:      in.SecretRef.Name,
		}
		if result.Namespace == "" {
			result.Namespace = defaultNamespace
		}
		return result, true
	}
	return apitypes.NamespacedName{}, false
}

// GetValuesYaml 优先使用secret，如果不存在尝试使用yaml inline.
func (in *YAMLText) GetValuesYaml(ctx context.Context, c client.Client, defaultNamespace string) (string, error) {
	if in.IsEmpty() {
		return "", nil
	}
	secretKey, ok := in.getAddonSecretNamespacedName(defaultNamespace)
	if ok {
		var err error
		secret := &corev1.Secret{}
		if err = c.Get(ctx, secretKey, secret); err != nil && !apierrors.IsNotFound(err) {
			return "", err
		}
		return string(secret.Data[valuesYamlKey]), nil
	} else {
		return in.Content, nil
	}
}

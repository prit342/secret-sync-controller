/*
Copyright 2025.

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

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1alpha1 "github.com/prit342/secret-sync-controller/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
)

// mapSecretToSecretSyncs maps a Secret object event to one or more SecretSync reconcile requests.
// This is used when a Secret is created/updated, to find all SecretSync CRs that reference it
// as their source secret, so that the controller can re-sync them.
func (r *SecretSyncReconciler) mapSecretToSecretSyncs(ctx context.Context, obj client.Object) []ctrl.Request {
	srcSecret, ok := obj.(*corev1.Secret)
	if !ok {
		// this is not a secret, so we don't care about this watch event, ideally this should not happen
		return nil
	}

	var (
		reqs           []ctrl.Request              // list of reconcile requests to be returned
		syncSecretList syncv1alpha1.SecretSyncList // list of all SecretSync CRs in the cluster
	)
	// list all the CRs of type SecretSync in the cluster
	if err := r.List(ctx, &syncSecretList); err != nil {
		return nil
	}

	// iterate over all the SecretSync CRs and check if the the secret that was watched
	// matches the source secret of any of the SecretSync CRs
	// if it does, we need to requeue the reconcile request for that CR
	// this is because the source secret has changed and we need to re-sync it
	// to the target namespaces
	for _, syncSecret := range syncSecretList.Items {
		if syncSecret.Spec.SourceName == srcSecret.Name &&
			syncSecret.Spec.SourceNamespace == srcSecret.Namespace {
			// we found a match, so we need to requeue the reconcile request for this CR
			reqs = append(reqs, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      syncSecret.Name,
					Namespace: syncSecret.Namespace,
				},
			})
		}
	}
	return reqs

}

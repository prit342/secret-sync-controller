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

// mapSecretToSecretSyncs maps a Secret event to one or more SecretSync reconcile requests.
// This function is called when any Secret is created or updated.
//
// It uses the field index "bySourceSecret" (sourceNamespace/sourceName) to efficiently find
// all SecretSync custom resources that reference the changed Secret as their source.
//
// For example:
//   - A Secret named "example-secret" in "test-source" is updated
//   - This function finds all SecretSyncs with:
//     spec.sourceName: "example-secret"
//     spec.sourceNamespace: "test-source"
//   - Each matching SecretSync is then re-queued for reconciliation to re-sync the source to targets
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

	// now we look into our index and see if there are CRs that point this source secret
	// we have created this index already
	// from our cache we try to obtain all the CRs that point to the source secret
	if err := r.List(ctx, &syncSecretList, client.MatchingFields{
		bySourceSecretIndexKey: srcSecret.Name + "/" + srcSecret.Namespace,
	}); err != nil {
		return nil // on error do not requeue
	}
	// iterate through all the syncSecrets that the secret that we watched points to
	for _, syncSecret := range syncSecretList.Items {
		reqs = append(reqs, ctrl.Request{NamespacedName: types.NamespacedName{
			Name:      syncSecret.Name,
			Namespace: syncSecret.Namespace,
		}})
	}

	return reqs
}

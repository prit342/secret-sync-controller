package controller

import (
	"context"
	"errors"
	"fmt"

	syncv1alpha1 "github.com/prit342/secret-sync-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// deleteChildObjects - deletes the child objects that belongs to the instance i.e  CR
func (r *SecretSyncReconciler) deleteChildObjects(
	ctx context.Context, // context for the API call
	instance *syncv1alpha1.SecretSync, // the CR that should create the secrets
) error {

	var combineErr error
	for _, ns := range instance.Spec.TargetNamespaces {
		// delete the secret in the target namespace
		err := r.Delete(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.Spec.SourceName,
				Namespace: ns,
			},
		})

		// we ignore the not found error, if the object does not exist it means we don't need to delete it
		// we just want to delete the object if it exists
		if err != nil && client.IgnoreNotFound(err) != nil {
			combineErr = errors.Join(combineErr, fmt.Errorf("error deleting secret %s in namespace %s: %w",
				instance.Spec.SourceName, ns, err))
		}
	}
	return combineErr
}

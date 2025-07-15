package controller

import (
	"context"
	"fmt"

	syncv1alpha1 "github.com/prit342/secret-sync-controller/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// RemoveFinalizer - removes the finalizer from the CR
func (r *SecretSyncReconciler) RemoveFinalizer(
	_ context.Context, // context for the API call
	instance *syncv1alpha1.SecretSync, // the CR that called the reconcile function
	finalizer string, // the finalizer to be removed
) error {

	if !controllerutil.ContainsFinalizer(instance, finalizer) {
		return fmt.Errorf("finalizer %q not found in Custom resource %q in namespace %q",
			finalizer, instance.Name, instance.Namespace)
	}

	if ok := controllerutil.RemoveFinalizer(instance, secretSyncFinalizer); !ok {
		return fmt.Errorf("failed to remove finalizer %q from %q in namespace %q",
			finalizer, instance.Name, instance.Namespace)
	}

	return nil
}

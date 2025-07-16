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
	"errors"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	syncv1alpha1 "github.com/prit342/secret-sync-controller/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
)

// SecretSyncReconciler reconciles a SecretSync object
type SecretSyncReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	// controllerName is the name of the controller
	annotationsLen = 3 // number of annotations we will add to the copied object
	//
	controllerNameKey   = "app.kubernetes.io/managed-by"
	controllerNameValue = "secret-sync-controller"
	//
	controllerOwnerNameKey      = "secretsync.example.com/owner-name"
	controllerOwnerNamespacekey = "secretsync.example.com/owner-namespace"
	secretSyncFinalizer         = "secretsync.example.com/finalizer" // finalizer to be added to the SecretSync object
	requeueDelay                = 7 * time.Minute
)

// +kubebuilder:rbac:groups=sync.example.com,resources=secretsyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sync.example.com,resources=secretsyncs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sync.example.com,resources=secretsyncs/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SecretSync object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *SecretSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// l := logf.FromContext(ctx)
	l := logf.Log.WithName("SecretSyncReconciler")

	l.Info("reconciling for", req.Name, req.NamespacedName)
	// instance is the CR that called the reconcile function
	instance := &syncv1alpha1.SecretSync{}

	// step 1: fetch the instance from the API server
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if client.IgnoreNotFound(err) == nil {
			l.Info("instance not found, it might have been deleted", "name", req.Name, "namespace", req.Namespace)
			return ctrl.Result{}, nil // No need to requeue, the instance is not present
		}
		l.Error(err, "failed to get instance", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, err // ignore not found errors
	}

	// check if the instance has our finalizer
	objectHasFinalizer := controllerutil.ContainsFinalizer(instance, secretSyncFinalizer)

	// Step 2: Handle deletion
	if !instance.DeletionTimestamp.IsZero() { // CR is marked for deletion
		// Handle cleanup of synced secrets
		// Remove finalizer, return
		l.Info("deleting instance and child resources", "name", instance.Name, "namespace", instance.Namespace)
		if err := r.deleteChildObjects(ctx, instance); err != nil {
			l.Error(err, "failed to delete child objects")
			if err := r.updateStatus(ctx, instance, 
				fmt.Sprintf("failed to delete child objects: %s", err), true); err != nil {
				l.Error(err, "failed to update status after error deleting child objects")
				}
			}
			// we want to requeue this request to try again later
			return ctrl.Result{RequeueAfter: requeueDelay}, nil // Try again later
		}
		// remove finzalizer if it exists
		if err := r.RemoveFinalizer(ctx, instance, secretSyncFinalizer); err != nil {
			l.Error(err, "failed to delete finalizer")
			if err := r.updateStatus(ctx, instance, 
				fmt.Sprintf("%s", err), true); err != nil { // Update status with error
				l.Error(err, "failed to update status after error removing finalizer")
				// we want to requeue this request to try again later
				}
			return ctrl.Result{RequeueAfter: requeueDelay}, nil      // Try again later
		}

		if err := r.Update(ctx, instance); err != nil {
			l.Error(err, "failed to update instance after removing finalizer")
			return ctrl.Result{RequeueAfter: requeueDelay}, nil // Try again later
		}
		l.Info("finalizer removed and child resources deleted", "name", instance.Name, "namespace", instance.Namespace)
		return ctrl.Result{}, nil // No need to requeue, cleanup done

	}
	// Step 3: Add finalizer if not present
	// on the first reconciliation, we need to add the finalizer
	// this is to ensure that we can clean up the child resources when the CR is deleted
	if !objectHasFinalizer {
		if ok := controllerutil.AddFinalizer(instance, secretSyncFinalizer); !ok {
			l.Error(fmt.Errorf("failed to add finalizer %s to instance %s", secretSyncFinalizer, instance.Name),
				"failed to add finalizer")
			return ctrl.Result{RequeueAfter: 7 * time.Minute}, nil // Try again later
		}
		if err := r.Update(ctx, instance); err != nil {
			l.Error(err, "failed to update instance after adding finalizer")
			return ctrl.Result{RequeueAfter: requeueDelay}, nil // Try again later
		}
		// Return to requeue and ensure consistent state before continuing
		// we have added the finalizer, so we can requeue
		l.Info("finalizer added to instance", "name", instance.Name, "namespace", instance.Namespace)
		return ctrl.Result{Requeue: true}, nil
	}

	// check if the source namespace is also part of the destination namespace
	// we cannot copy the source object to itself
	if err := checkSourceInTargetNamespaces(instance); err != nil {
		l.Error(err, "failed to sync")
		if uerr := r.updateStatus(ctx, instance, err.Error(), true); uerr != nil {
			return ctrl.Result{}, errors.Join(uerr, err) // Try again later
		}
		return ctrl.Result{}, nil // No need to requeue, we have updated the status
	}
	//
	srcSecret := &corev1.Secret{} // the source secret we need to sync/copy to the target namespaces
	// try to read the source secret from the source namespace
	if err := r.Get(ctx, types.NamespacedName{
		Name:      instance.Spec.SourceName,
		Namespace: instance.Spec.SourceNamespace,
	}, srcSecret); err != nil {
		// if there was an error reading the source secret, we will update the status
		// with correct message and requeue and retry later
		msg := fmt.Sprintf("error reading source secret %s in namespace %s: %s",
			instance.Spec.SourceName, instance.Spec.SourceNamespace, err)
		l.Error(err, msg)
		if err := r.updateStatus(ctx, instance, msg, true); err != nil {
			l.Error(err, "failed to update status after error reading source secret")
			return ctrl.Result{RequeueAfter: requeueDelay}, nil // Try again later after 7 minutes
		}
		return ctrl.Result{RequeueAfter: requeueDelay}, nil // No need to requeue, we have updated the status
	}

	// sync the object into the target namespaces
	if err := r.syncSecretToNamespaces(ctx, instance, srcSecret, instance.Spec.TargetNamespaces); err != nil {
		l.Error(err, "failed to copy the source secret to destination namespaces")
		if uerr := r.updateStatus(ctx, instance, fmt.Sprintf("failed to sync object: %s", err), true); uerr != nil {
			l.Error(uerr, "failed to update status after sync error")
			return ctrl.Result{RequeueAfter: requeueDelay}, nil // Try again later after 7 minutes
		}
		return ctrl.Result{RequeueAfter: requeueDelay}, nil // we cannot continue, try again later
	}

	// once synced, we need to update the status
	successMessage := fmt.Sprintf("successfully synced secret %s to namespaces: %s",
		instance.Spec.SourceName, strings.Join(instance.Spec.TargetNamespaces, ","))
	if err := r.updateStatus(ctx, instance, successMessage, false); err != nil {
		l.Error(err, "failed to update status after successful sync")
		return ctrl.Result{RequeueAfter: requeueDelay}, nil // Try again later after 7 minutes
	}

	l.Info(successMessage)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Primary resource: Reconcile will be triggered when a SecretSync object is created, updated, or deleted.
		For(&syncv1alpha1.SecretSync{}).
		//
		// Secondary watch: also watch Secret objects across all namespaces.
		// This ensures we re-trigger reconciliation of relevant SecretSyncs if the source Secret is created or updated.
		//
		// For example:
		// - If a SecretSync exists but the source Secret doesn't yet, this will allow us to sync it once it appears.
		// - If a Secret is updated (e.g., its data changes), we want to propagate the changes to all target namespaces.
		Watches(
			&corev1.Secret{}, // The object type to watch
			//
			// The mapping function: when a Secret is changed, we call this function
			// to determine which SecretSyncs reference this Secret (via sourceName + sourceNamespace).
			handler.EnqueueRequestsFromMapFunc(r.mapSecretToSecretSyncs),
			//
			// Predicate: limit the watch to only fire when a meaningful change happens to the Secret.
			// This filters out no-op updates. For example, this triggers on:
			// - Secret creation
			// - Secret data changes
			// - Label/annotation updates
			// But NOT on resyncs with no changes
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Named("secretsync"). // Give the controller a name for logs/metrics/etc.
		Complete(r)          // Complete the controller setup
}

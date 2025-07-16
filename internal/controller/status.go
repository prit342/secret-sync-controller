package controller

import (
	"context"
	"fmt"
	"strings"

	syncv1alpha1 "github.com/prit342/secret-sync-controller/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	successReason = "SecretSyncedSuccessfully"
	failedReason  = "SecretSyncFailed"
)

// updateStatus - udpates the status of the CR object
func (r *SecretSyncReconciler) updateStatus(
	ctx context.Context, // context for the API call
	instance *syncv1alpha1.SecretSync, // the CR that needs to be updated
	message string, // message on the status field
	isError bool, // if true, the status will be set to "Error", otherwise "Success"
) error {
	// we set some fields to reflect that current state of the CR
	instance.Status.LastSyncTime = metav1.Now()

	// Set condition
	condition := metav1.Condition{
		Type:               "Synced", // type of the condition
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: instance.Generation,
		Message:            message,
		Status:             metav1.ConditionTrue,
		Reason:             successReason,
	}
	// if there is an error, we set the status to "False"
	if isError {
		condition.Status = metav1.ConditionFalse
		condition.Reason = failedReason
	}

	// we are not appending the condition, we are replacing it
	// this is because we want to have only one condition of type "Synced" at a time
	instance.Status.Conditions = []metav1.Condition{condition}
	return r.Status().Update(ctx, instance)
}

// checkSourceInTargetNamespaces checks if the source namespace is part of the target namespaces
func checkSourceInTargetNamespaces(instance *syncv1alpha1.SecretSync) error {
	for _, ns := range instance.Spec.TargetNamespaces {
		if ns == instance.Spec.SourceNamespace {
			return fmt.Errorf("the sourceNamespace %s is in the targetNamespaces list %s, please remove this",
				ns, strings.Join(instance.Spec.TargetNamespaces, ","))
		}
	}
	return nil
}

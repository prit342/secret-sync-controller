package controller

import (
	"context"
	"errors"
	"fmt"

	syncv1alpha1 "github.com/prit342/secret-sync-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// syncSecretToNamespaces - copies src secret to the dst namespaces
func (r *SecretSyncReconciler) syncSecretToNamespaces(
	ctx context.Context,
	instance *syncv1alpha1.SecretSync,
	srcSecret *corev1.Secret, // the source secret object
	dstNamespaces []string) error {
	// set the ower reference for the source object

	var combineErr error

	// get the type of the object that needs to be copied
	for _, ns := range dstNamespaces {

		// copy := srcObj.DeepCopyObject()
		copySecret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: srcSecret.APIVersion,
				Kind:       srcSecret.Kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      srcSecret.Name,
				Namespace: ns,
			},
			Data: srcSecret.Data,
			Type: srcSecret.Type,
		}

		// we cannot set the owner reference here because the object is being copied to a different namespace
		// and the owner reference is not allowed to be set across namespaces
		// we can set the owner reference only if the object is in the same namespace as the owner object
		// if err := controllerutil.SetControllerReference(instance, copy, r.Scheme); err != nil {
		// 	combineErr = errors.Join(err)
		// 	continue
		// }
		// so the above snippet does not work
		//
		// before we copy the object, we need to check if the secret exists in the target namespace
		if err := r.checkIfSecretAlreadyExistsAndNotOwned(ctx, instance, ns); err != nil {
			// if the secret already exists in the target namespace and is not owned by this CR,
			// we need to return an error and not copy the secret object
			// but we need to continue the loop so that we can check the next namespace
			combineErr = errors.Join(combineErr, err)
			continue
		}

		// we need to see the correct annotations and labels that we need to add to the copied object
		// we will add the controller name, owner name and owner namespace to the annotations
		// we will also add the controller name to the labels
		// to the copy object
		annotations := make(map[string]string, annotationsLen)
		annotations[controllerNameKey] = controllerNameValue
		annotations[controllerOwnerNameKey] = instance.Name
		annotations[controllerOwnerNamespacekey] = instance.Namespace

		// we are setting both labels and annotations to the copied object
		// this is because we want to be able to filter the objects based on the labels
		// and also want to be able to find the owner of the object based on the annotations
		// this is useful when we want to delete the object later
		// for example, if we want to delete the object later, we can filter the objects based on the labels
		// and then delete the objects that have the same owner name
		// and owner namespace as the instance
		// this way we can delete all the objects that are owned by this instance
		// we can also use the annotations to find the owner of the object
		copySecret.SetLabels(annotations)
		copySecret.SetAnnotations(annotations)
		patchErr := r.Patch(ctx, copySecret, client.Apply, client.FieldOwner(controllerNameValue))
		combineErr = errors.Join(patchErr)
	}

	return combineErr
}

// checkIfSecretAlreadyExists checks if the secret or configmap already exists in the target namespace
// if this is the case, we need to check the annotations on the child object i.e the secret or configmap
// based on the annotations, we can decide if the object is owned by this CR or not
// If the object is owned by this CR, we can continue
// If the object is not owned by this CR, we return an error
// this error exists because we are copying the secret as it is and not generating a new name for the secret
// for example, if the source secret is called foo then the destiation secret will also be called foo but
// will be in a different namespace
func (r *SecretSyncReconciler) checkIfSecretAlreadyExistsAndNotOwned(
	ctx context.Context, // context for the API call
	instance *syncv1alpha1.SecretSync, // the CR that called the reconcile function
	ns string, // the namespace where we want to check if the secret already exists
) error {

	var secret corev1.Secret // this is where we will store the secret object we read from

	// read the object from our local cache to see if already exists
	err := r.Get(ctx, types.NamespacedName{
		Name:      instance.Spec.SourceName, // the name of the secret
		Namespace: ns,                       // the namespace where we want to check
	}, &secret)

	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			// if the object is not found, we can continue
			return nil // this means that the object does not exist in the target namespace
		}
		return fmt.Errorf("error reading object %s in namespace %s: %w", instance.Spec.SourceName, ns, err)
	}
	// so the object already exists in the target namespace
	// we need to check if the object is owned by this CR or not
	// we will check the annotations on the secret to see if it is owned by this CR or not
	// if the object is owned by this CR, we can continue
	// if the object is not owned by this CR, we need to return an error
	annots := secret.GetAnnotations()
	// if the object has no annotations, we need to return an error
	// this is because we cannot determine if the object is owned by this CR or not
	if annots == nil {
		// if there are no annotations, we can continue
		return fmt.Errorf("the secret %s already exists in namespace %s but has no"+
			" annotations, please check if this is owned by this CR", instance.Spec.SourceName, ns)
	}
	// if the object is not owned by this CR, we need to return an error as we cannot copy the object
	// it might be owned by another CR or it might be a manually created object
	if val, ok := annots[controllerNameKey]; ok && val != controllerNameValue {
		return fmt.Errorf("the secret %s already exists in namespace %s and is not owned by this CR, "+
			"please check if this is owned by this CR", instance.Spec.SourceName, ns)
	}
	// at this stage, we know that the object is owned by an instance of this controller
	// but we need to check if the object is owned by this particular instance of the controller
	// we will check the annotations on the secret to see if it is owned by this instance or not
	// if the object is NOT owned by this particular instance, we need to return an error
	if val, ok := annots[controllerOwnerNameKey]; ok && val != instance.Name {
		return fmt.Errorf("the secret %s already exists in namespace %s and is not owned by this instance %s",
			instance.Spec.SourceName, ns, instance.Name)
	}
	// Finally we also check if the namespace of the owner is the same as the instance namespace
	// if the namespace of the owner is not the same as the instance namespace, we
	// need to return an error as we cannot copy the object
	// this is because the owner namespace is not the same as the instance namespace
	if val, ok := annots[controllerOwnerNamespacekey]; ok && val != instance.Namespace {
		return fmt.Errorf("the secret %s already exists in namespace %s and is not owned by this instance %s, "+
			"please check if this is owned by this instance", instance.Spec.SourceName, ns, instance.Name)
	}

	return nil // this means that the object is owned by this instance and we can continue
}

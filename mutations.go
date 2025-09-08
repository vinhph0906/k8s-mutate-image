package main

import (
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	podResource         = metav1.GroupVersionResource{Version: "v1", Resource: "pods"}
	volumeClaimResource = metav1.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}
)

// applyMutations implements the logic of our admission controller webhook.
func (wh *mutationWH) applyMutations(req *admissionv1.AdmissionRequest) ([]patchOperation, error) {
	// This handler should only get called on Pod or Pvc objects as per the MutatingWebhookConfiguration in the YAML file.
	// However, if (for whatever reason) this gets invoked on an object of a different kind, issue a log message but
	// let the object request pass through otherwise.
	switch req.Resource {
	case podResource:
		// Parse the Pod object.
		raw := req.Object.Raw
		pod := corev1.Pod{}
		if _, _, err := universalDeserializer.Decode(raw, nil, &pod); err != nil {
			return nil, fmt.Errorf("could not deserialize pod object: %v", err)
		}

		return wh.applyMutationOnPod(pod)

	case volumeClaimResource:
		// Parse the Pvc object.
		raw := req.Object.Raw
		pvc := corev1.PersistentVolumeClaim{}
		if _, _, err := universalDeserializer.Decode(raw, nil, &pvc); err != nil {
			return nil, fmt.Errorf("could not deserialize pvc object: %v", err)
		}

		return wh.applyMutationOnPvc(pvc)
	default:
		wh.logger.WithField("resource", req.Resource).Warn("Got an unexpected resource, don't know what to do with...")
	}
	return nil, nil
}

// applyMutationOnPod gets the deserialized pod spec and returns the patch operations
// to apply, if any, or an error if something went wrong.
func (wh *mutationWH) applyMutationOnPod(pod corev1.Pod) ([]patchOperation, error) {

	var patches []patchOperation

	if wh.registries != nil {
		patches = append(patches, wh.patchRegistry(pod)...)
	}

	if wh.forceImagePullPolicy {
		patches = append(patches, wh.patchImagePullPolicy(pod)...)
	}

	if wh.imagePullSecret != "" {
		patches = append(patches, wh.patchImagePullSecret(pod)...)
	}

	wh.logger.Debugf("Patch applied: %v", patches)

	return patches, nil
}

// applyMutationOnPvc gets the deserialized pvc spec and returns the patch operations
// to apply, if any, or an error if something went wrong.
func (wh *mutationWH) applyMutationOnPvc(pvc corev1.PersistentVolumeClaim) ([]patchOperation, error) {

	var patches []patchOperation

	if wh.defaultStorageClass != "" {
		if pvc.Spec.StorageClassName != nil {
			if *pvc.Spec.StorageClassName != wh.defaultStorageClass {
				patches = append(patches, patchOperation{
					Op:    "replace",
					Path:  "/spec/storageClassName",
					Value: wh.defaultStorageClass,
				})
			}
		} else {
			patches = append(patches, patchOperation{
				Op:    "add",
				Path:  "/spec/storageClassName",
				Value: wh.defaultStorageClass,
			})
		}
	}

	wh.logger.Debugf("Patch applied: %v", patches)

	return patches, nil
}

package main

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func (wh *mutationWH) patchRegistry(pod corev1.Pod) []patchOperation {
	var patches []patchOperation
	if pod.Spec.InitContainers != nil {
		for i, c := range pod.Spec.InitContainers {
			wh.logger.Tracef("/spec/initContainers/%d/image = %s", i, c.Image)
			newImage, ok := wh.replaceRegistryIfSet(c.Image)
			if ok {
				patches = append(patches, patchOperation{
					Op:    "replace",
					Path:  fmt.Sprintf("/spec/initContainers/%d/image", i),
					Value: newImage,
				})
			}
			wh.logger.Infof("replace /spec/initContainers/%d/image = %s", i, newImage)
		}
	}

	if pod.Spec.Containers != nil {
		for i, c := range pod.Spec.Containers {
			wh.logger.Tracef("/spec/containers/%d/image = %s", i, c.Image)

			newImage, ok := wh.replaceRegistryIfSet(c.Image)
			if ok {
				patches = append(patches, patchOperation{
					Op:    "replace",
					Path:  fmt.Sprintf("/spec/containers/%d/image", i),
					Value: newImage,
				})
			}
			wh.logger.Infof("replace /spec/containers/%d/image = %s", i, newImage)
		}
	}
	return patches
}
func (wh *mutationWH) patchImagePullPolicy(pod corev1.Pod) []patchOperation {
	var patches []patchOperation
	if pod.Spec.InitContainers != nil {
		for i, c := range pod.Spec.InitContainers {
			wh.logger.Tracef("/spec/initContainers/%d/imagePullPolicy = %s", i, c.ImagePullPolicy)
			op := "replace"
			// still take the case when ImagePullPolicy is empty, but this case should not happen.
			// Policy defaults to Always if tag is latest, IfNotPresent otherwise.
			if c.ImagePullPolicy == "" {
				op = "add"
			}
			if wh.imagePullPolicyToForce != c.ImagePullPolicy {
				patches = append(patches, patchOperation{
					Op:    op,
					Path:  fmt.Sprintf("/spec/initContainers/%d/imagePullPolicy", i),
					Value: wh.imagePullPolicyToForce,
				})
			}
		}
	}

	if pod.Spec.Containers != nil {
		for i, c := range pod.Spec.Containers {
			wh.logger.Tracef("/spec/containers/%d/imagePullPolicy = %s", i, c.ImagePullPolicy)
			if c.ImagePullPolicy != wh.imagePullPolicyToForce {
				op := "replace"
				if c.ImagePullPolicy == "" {
					op = "add"
				}
				patches = append(patches, patchOperation{
					Op:    op,
					Path:  fmt.Sprintf("/spec/containers/%d/imagePullPolicy", i),
					Value: wh.imagePullPolicyToForce,
				})
			}
		}
	}
	return patches
}
func (wh *mutationWH) patchImagePullSecret(pod corev1.Pod) []patchOperation {
	var patches []patchOperation
	// if there are no existing pull secrets, append or replace is the same operation.
	if pod.Spec.ImagePullSecrets == nil {
		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/spec/imagePullSecrets",
			Value: []map[string]string{{"name": wh.imagePullSecret}},
		})
	} else {
		if wh.appendImagePullSecret {
			// in the append branch,
			// in case of existing secrets in the pod, we check if the secret does not exist and we append it to the list
			imagePullSecretsAlreadyExist := false
			for _, i := range pod.Spec.ImagePullSecrets {
				if i.Name == wh.imagePullSecret {
					imagePullSecretsAlreadyExist = true
					break
				}
			}
			if !imagePullSecretsAlreadyExist {
				patches = append(patches, patchOperation{
					Op:    "add",
					Path:  fmt.Sprintf("/spec/imagePullSecrets/%d", len(pod.Spec.ImagePullSecrets)),
					Value: []map[string]string{{"name": wh.imagePullSecret}},
				})
			}
		} else {
			// in the replace branch,
			// if the secret is not the one to set, we replace the existing secret(s)
			if !(len(pod.Spec.ImagePullSecrets) == 1 && pod.Spec.ImagePullSecrets[0].Name == wh.imagePullSecret) {
				patches = append(patches, patchOperation{
					Op:    "replace",
					Path:  "/spec/imagePullSecrets",
					Value: []map[string]string{{"name": wh.imagePullSecret}},
				})
			}
		}
	}
	return patches
}

// replaceRegistryIfSet assumes the image format is a.b[:port]/c/d:e
// if a.b is present, it is replaced by the registry given as argument.
func (wh *mutationWH) replaceRegistryIfSet(image string) (string, bool) {

	imageParts := strings.Split(image, "/")
	if len(imageParts) == 1 {
		// case imagename or imagename:version, where version can contains .
		registry, ok := wh.registries["default"]
		if !ok {
			return "", false
		}
		imageParts = append([]string{registry}, imageParts...)
	} else {
		// case something/imagename:version, assessing the something part.
		if strings.Contains(imageParts[0], ".") {
			registry, ok := wh.registries[imageParts[0]]
			if !ok {
				return "", false
			}

			imageParts[0] = registry
		} else {
			registry, ok := wh.registries["default"]
			if !ok {
				return "", false
			}
			imageParts = append([]string{registry}, imageParts...)
		}
	}

	return strings.Join(imageParts, "/"), true
}

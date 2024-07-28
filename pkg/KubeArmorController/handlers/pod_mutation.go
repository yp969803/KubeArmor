// SPDX-License-Identifier: Apache-2.0
// Copyright 2022 Authors of KubeArmor

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	"golang.org/x/mod/semver"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// PodAnnotator Structure
type PodAnnotator struct {
	Client    client.Client
	Decoder   *admission.Decoder
	Logger    logr.Logger
	Enforcer  string
	K8Version string
}

const k8sVisibility = "process,file,network,capabilities"
const appArmorAnnotation = "container.apparmor.security.beta.kubernetes.io/"

// +kubebuilder:webhook:path=/mutate-pods,mutating=true,failurePolicy=Ignore,groups="",resources=pods,verbs=create;update,versions=v1,name=annotation.kubearmor.com,admissionReviewVersions=v1,sideEffects=NoneOnDryRun

// Handle Pod Annotation
func (a *PodAnnotator) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	if err := a.Decoder.Decode(req, pod); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Decode will omit sometimes the namespace value for some reason copying it manually
	if pod.Namespace == "" {
		pod.Namespace = req.Namespace
	}

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	// == Policy == //

	if _, ok := pod.Annotations["kubearmor-policy"]; !ok {
		// if no annotation is set enable kubearmor by default
		pod.Annotations["kubearmor-policy"] = "enabled"
	} else if pod.Annotations["kubearmor-policy"] != "enabled" && pod.Annotations["kubearmor-policy"] != "disabled" && pod.Annotations["kubearmor-policy"] != "audited" {
		// if kubearmor policy is not set correctly, default it to enabled
		pod.Annotations["kubearmor-policy"] = "enabled"
	}

	// == LSM == //

	if a.Enforcer == "AppArmor" {
		appArmorAnnotator(pod, a.K8Version)
	}

	// == Exception == //

	// exception: kubernetes app
	if pod.Namespace == "kube-system" {
		if _, ok := pod.Labels["k8s-app"]; ok {
			pod.Annotations["kubearmor-policy"] = "audited"
		}

		if value, ok := pod.Labels["component"]; ok {
			if value == "etcd" || value == "kube-apiserver" || value == "kube-controller-manager" || value == "kube-scheduler" {
				pod.Annotations["kubearmor-policy"] = "audited"
			}
		}
	}

	// exception: cilium-operator
	if _, ok := pod.Labels["io.cilium/app"]; ok {
		pod.Annotations["kubearmor-policy"] = "audited"
	}

	// exception: kubearmor
	if _, ok := pod.Labels["kubearmor-app"]; ok {
		pod.Annotations["kubearmor-policy"] = "audited"
	}

	// == Visibility == //

	if _, ok := pod.Annotations["kubearmor-visibility"]; !ok {
		pod.Annotations["kubearmor-visibility"] = k8sVisibility
	}

	// == //

	// send the mutation response
	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// == Add AppArmor annotations == //
func appArmorAnnotator(pod *corev1.Pod, k8Version string) {
	podAnnotations := map[string]string{}
	var podOwnerName string

	// podOwnerName is the pod name for static pods and parent object's name
	// in other cases
	for _, ownerRef := range pod.ObjectMeta.OwnerReferences {
		// pod is owned by a replicaset, daemonset etc thus we use the managing
		// controller's name
		if *ownerRef.Controller {
			podOwnerName = ownerRef.Name

			if ownerRef.Kind == "ReplicaSet" {
				// if it belongs to a replicaset, we also remove the pod template hash
				podOwnerName = strings.TrimSuffix(podOwnerName, fmt.Sprintf("-%s", pod.ObjectMeta.Labels["pod-template-hash"]))
			}
		}
	}

	if podOwnerName == "" {
		// pod is standalone, name remains constant
		podOwnerName = pod.ObjectMeta.Name
	}

	// Check if the k8 version >= 1.30
	k8VerGreater := isVersionGreaterThanOrEqual(k8Version, "v1.30")

	// Get existant kubearmor annotations
	for k, v := range pod.Annotations {

		if strings.HasPrefix(k, appArmorAnnotation) {

			if v == "unconfined" {
				containerName := strings.Split(k, "/")[1]
				podAnnotations[containerName] = v
			} else {
				containerName := strings.Split(k, "/")[1]
				podAnnotations[containerName] = strings.Split(v, "/")[1]
			}
			if k8VerGreater {
				// Remove appArmorAnnotation k8s 1.30 compatiblity issue
				delete(pod.Annotations, k)
			}
		}
	}

	for _, c := range pod.Spec.Containers {

		if k8VerGreater {
			if (pod.Spec.SecurityContext == nil || pod.Spec.SecurityContext.AppArmorProfile == nil) && (c.SecurityContext == nil || c.SecurityContext.AppArmorProfile == nil) {
				if v, ok := podAnnotations[c.Name]; !ok {

					profile := "kubearmor-" + pod.Namespace + "-" + podOwnerName + "-" + c.Name

					c.SecurityContext.AppArmorProfile = &corev1.AppArmorProfile{
						Type:             corev1.AppArmorProfileTypeLocalhost,
						LocalhostProfile: ptr.To(profile),
					}
				} else {

					if v == "unconfined" {
						c.SecurityContext.AppArmorProfile = &corev1.AppArmorProfile{
							Type: corev1.AppArmorProfileTypeUnconfined,
						}
					} else {

						c.SecurityContext.AppArmorProfile = &corev1.AppArmorProfile{
							Type:             corev1.AppArmorProfileTypeLocalhost,
							LocalhostProfile: ptr.To(v),
						}
					}
				}
			}
		} else {
			if _, ok := podAnnotations[c.Name]; !ok {
				podAnnotations[c.Name] = "kubearmor-" + pod.Namespace + "-" + podOwnerName + "-" + c.Name
			}
		}
	}

	if k8VerGreater {
		// Add kubearmor annotations to the pod
		for k, v := range podAnnotations {
			if v == "unconfined" {
				continue
			}
			pod.Annotations[appArmorAnnotation+k] = "localhost/" + v
		}
	}
}

func isVersionGreaterThanOrEqual(v1, v2 string) bool {
	return semver.Compare(v1, v2) >= 0
}

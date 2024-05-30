// SPDX-License-Identifier: Apache-2.0
// Copyright 2022 Authors of KubeArmor

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	"context"
	time "time"

	securitykubearmorcomv1 "github.com/kubearmor/KubeArmor/pkg/KubeArmorController/api/security.kubearmor.com/v1"
	versioned "github.com/kubearmor/KubeArmor/pkg/KubeArmorController/client/clientset/versioned"
	internalinterfaces "github.com/kubearmor/KubeArmor/pkg/KubeArmorController/client/informers/externalversions/internalinterfaces"
	v1 "github.com/kubearmor/KubeArmor/pkg/KubeArmorController/client/listers/security.kubearmor.com/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// KubeArmorClusterPolicyInformer provides access to a shared informer and lister for
// KubeArmorClusterPolicies.
type KubeArmorClusterPolicyInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.KubeArmorClusterPolicyLister
}

type kubeArmorClusterPolicyInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewKubeArmorClusterPolicyInformer constructs a new informer for KubeArmorClusterPolicy type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewKubeArmorClusterPolicyInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredKubeArmorClusterPolicyInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredKubeArmorClusterPolicyInformer constructs a new informer for KubeArmorClusterPolicy type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredKubeArmorClusterPolicyInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.SecurityV1().KubeArmorClusterPolicies().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.SecurityV1().KubeArmorClusterPolicies().Watch(context.TODO(), options)
			},
		},
		&securitykubearmorcomv1.KubeArmorClusterPolicy{},
		resyncPeriod,
		indexers,
	)
}

func (f *kubeArmorClusterPolicyInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredKubeArmorClusterPolicyInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *kubeArmorClusterPolicyInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&securitykubearmorcomv1.KubeArmorClusterPolicy{}, f.defaultInformer)
}

func (f *kubeArmorClusterPolicyInformer) Lister() v1.KubeArmorClusterPolicyLister {
	return v1.NewKubeArmorClusterPolicyLister(f.Informer().GetIndexer())
}

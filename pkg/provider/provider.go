/*
Copyright 2023 Avi Zimmerman <avi.zimmerman@gmail.com>

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

// Package provider contains the storage provider implementation for Kubernetes.
package provider

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

// Options are the options for configuring the provider.
type Options struct {
	// Manager is the controller-runtime manager.
	Manager ctrl.Manager
	// Log is the controller-runtime logger.
	Log logr.Logger
}

// Provider is the storage provider implementation for Kubernetes.
type Provider struct {
	Options
	log logr.Logger
}

// New creates a new Provider.
func New(options Options) (*Provider, error) {
	p := &Provider{
		Options: options,
		log:     options.Log.WithName("storage-provider"),
	}
	err := ctrl.NewControllerManagedBy(options.Manager).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(p.enqueueObjectIfOwner)).
		Complete(p)
	if err != nil {
		p.log.Error(err, "Problem creating controller")
		return nil, err
	}
	return p, nil
}

// Start starts the provider. This is a blocking call that waits for the
// given context to be canceled.
func (p *Provider) Start(ctx context.Context) error {
	p.log.Info("Starting controller manager and storage provider")
	if err := p.Manager.Start(ctx); err != nil {
		p.log.Error(err, "Problem running controller manager")
		return err
	}
	return nil
}

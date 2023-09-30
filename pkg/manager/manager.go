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

// Package manager contains the controller-runtime manager.
package manager

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	corescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	storagev1 "github.com/webmeshproj/storage-provider-k8s/api/storage/v1"
)

// Manager is the controller-runtime manager.
type Manager = ctrl.Manager

// Options are the options for configuring the manager.
type Options struct {
	// WebhookPort is the address to bind the webhook server to.
	WebhookPort int
	// MetricsAddr is the address to bind the metrics endpoint to.
	MetricsPort int
	// ProbeAddr is the address to bind the health probe endpoint to.
	ProbePort int
	// ShutdownTimeout is the timeout for shutting down the manager.
	ShutdownTimeout time.Duration
	// DisableCache disables the controller-runtime cache. This is primarily
	// used for testing.
	DisableCache bool
}

// New returns a new controller-runtime manager.
func New(opts Options) (Manager, error) {
	return NewFromConfig(ctrl.GetConfigOrDie(), opts)
}

// NewFromConfig creates a new manager from the given config.
func NewFromConfig(cfg *rest.Config, opts Options) (Manager, error) {
	scheme := runtime.NewScheme()
	err := corescheme.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = storagev1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	probeAddr := "0"
	if opts.ProbePort != 0 {
		probeAddr = fmt.Sprintf("[::]:%d", opts.ProbePort)
	}
	metricsAddr := "0"
	if opts.MetricsPort != 0 {
		metricsAddr = fmt.Sprintf("[::]:%d", opts.MetricsPort)
	}
	mgropts := ctrl.Options{
		Scheme:                  scheme,
		GracefulShutdownTimeout: &opts.ShutdownTimeout,
		HealthProbeBindAddress:  probeAddr,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port: opts.WebhookPort,
		}),
		Controller: config.Controller{
			// Leader election is handled by the provider.
			NeedLeaderElection: &[]bool{false}[0],
		},
	}
	if opts.DisableCache {
		mgropts.Client = client.Options{
			Scheme: scheme,
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{&corev1.Secret{}},
			},
		}
	}
	mgr, err := ctrl.NewManager(cfg, mgropts)
	if err != nil {
		return nil, err
	}
	if opts.ProbePort != 0 {
		if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
			return nil, err
		}
		if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
			return nil, err
		}
	}
	return mgr, nil
}

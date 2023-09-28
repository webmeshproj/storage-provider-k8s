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

package provider

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/webmeshproj/storage-provider-k8s/pkg/manager"
)

func SetupTestProvider(t *testing.T) *Provider {
	testenv := envtest.Environment{
		UseExistingCluster:       &[]bool{true}[0],
		ControlPlaneStartTimeout: time.Second * 15,
		ControlPlaneStopTimeout:  time.Second * 15,
		AttachControlPlaneOutput: true,
	}
	cfg, err := testenv.Start()
	if err != nil {
		t.Fatal("Failed to start test environment", err)
	}
	mgr, err := manager.NewFromConfig(cfg, manager.Options{
		ShutdownTimeout: time.Second * 15,
	})
	if err != nil {
		t.Fatal("Failed to create manager", err)
	}
	provider, err := NewWithManager(mgr, Options{
		NodeID:                      uuid.NewString(),
		ListenAddr:                  "[::]:9443",
		Namespace:                   "default",
		LeaderElectionLeaseDuration: time.Second * 2,
		LeaderElectionRenewDeadline: time.Second * 2,
		LeaderElectionRetryPeriod:   time.Second * 1,
	})
	if err != nil {
		t.Fatal("Failed to create provider", err)
	}
	err = provider.Start(context.Background())
	if err != nil {
		t.Fatal("Failed to start provider", err)
	}
	return provider
}

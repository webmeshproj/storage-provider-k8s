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
	"fmt"
	"testing"
	"time"

	"github.com/webmeshproj/webmesh/pkg/storage"
	"github.com/webmeshproj/webmesh/pkg/storage/testutil"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/webmeshproj/storage-provider-k8s/provider/manager"
)

func TestProviderConformance(t *testing.T) {
	testutil.TestStorageProviderConformance(context.Background(), t, setupTestProviders)
}

func setupTestProviders(t *testing.T, count int) []storage.Provider {
	t.Log("Starting test environment")
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{Development: true})))
	testenv := envtest.Environment{
		ControlPlaneStartTimeout: time.Second * 30,
		ControlPlaneStopTimeout:  time.Second * 3,
	}
	cfg, err := testenv.Start()
	if err != nil {
		t.Fatal("Failed to start test environment", err)
	}
	t.Cleanup(func() {
		t.Log("Stopping test environment")
		err := testenv.Stop()
		if err != nil {
			t.Log("Failed to stop test environment", err)
		}
	})
	var providers []storage.Provider
	for i := 0; i < count; i++ {
		t.Log("Creating new controller manager", i)
		mgr, err := manager.NewFromConfig(cfg, manager.Options{
			ShutdownTimeout: time.Second * 3,
			DisableCache:    true,
			WebhookPort:     0,
			MetricsPort:     0,
			ProbePort:       0,
		})
		if err != nil {
			t.Fatal("Failed to create manager", i, err)
		}
		nodeID := fmt.Sprintf("test-node-%d", i)
		t.Log("Creating new provider", i, "ID:", nodeID)
		provider, err := NewWithManager(mgr, Options{
			NodeID:                      nodeID,
			ListenPort:                  9080 + i,
			Namespace:                   "default",
			LeaderElectionLeaseDuration: time.Second * 3,
			LeaderElectionRenewDeadline: time.Second * 1,
			LeaderElectionRetryPeriod:   time.Millisecond * 500,
		})
		if err != nil {
			t.Fatal("Failed to create provider", i, err)
		}
		providers = append(providers, provider)
	}
	return providers
}

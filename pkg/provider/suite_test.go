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
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/webmeshproj/webmesh/pkg/storage"
	"github.com/webmeshproj/webmesh/pkg/storage/testutil"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/webmeshproj/storage-provider-k8s/pkg/manager"
)

func TestProviderConformance(t *testing.T) {
	testutil.TestStorageProviderConformance(context.Background(), t, setupTestProvider)
}

var once sync.Once

func setupTestProvider(ctx context.Context, t *testing.T) storage.Provider {
	t.Log("Starting test environment")
	var useExisting bool
	testenv := envtest.Environment{
		ControlPlaneStartTimeout: time.Second * 15,
		ControlPlaneStopTimeout:  time.Second * 15,
		UseExistingCluster:       &useExisting,
	}
	cfg, err := testenv.Start()
	if err != nil {
		t.Fatal("Failed to start test environment", err)
	}
	once.Do(func() {
		useExisting = true
	})
	t.Cleanup(func() {
		err := testenv.Stop()
		if err != nil {
			t.Log("Failed to stop test environment", err)
		}
	})
	t.Log("Creating manager")
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
		LeaderElectionLeaseDuration: time.Second * 5,
		LeaderElectionRenewDeadline: time.Second * 3,
		LeaderElectionRetryPeriod:   time.Second * 1,
	})
	if err != nil {
		t.Fatal("Failed to create provider", err)
	}
	t.Log("Starting provider")
	err = provider.Start(context.Background())
	if err != nil {
		t.Fatal("Failed to start provider", err)
	}
	return provider
}

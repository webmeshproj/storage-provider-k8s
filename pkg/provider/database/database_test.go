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

package database

import (
	"context"
	"testing"
	"time"

	"github.com/webmeshproj/webmesh/pkg/storage"
	"github.com/webmeshproj/webmesh/pkg/storage/testutil"
	"github.com/webmeshproj/webmesh/pkg/storage/types"
	"k8s.io/apimachinery/pkg/runtime"
	corescheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	storagev1 "github.com/webmeshproj/storage-provider-k8s/api/storage/v1"
	"github.com/webmeshproj/storage-provider-k8s/pkg/manager"
)

func TestDatabaseConformance(t *testing.T) {
	testutil.TestMeshDBConformance(t, newTestDB)
}

func newTestDB(t *testing.T) (storage.MeshDB, types.PeerGraphStore) {
	t.Log("Starting test environment")
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{Development: true})))
	scheme := runtime.NewScheme()
	err := corescheme.AddToScheme(scheme)
	if err != nil {
		t.Fatal("Failed to add core scheme to runtime scheme:", err)
	}
	err = storagev1.AddToScheme(scheme)
	if err != nil {
		t.Fatal("Failed to add storage scheme to runtime scheme:", err)
	}
	testenv := envtest.Environment{
		Scheme:                   scheme,
		ErrorIfCRDPathMissing:    true,
		CRDDirectoryPaths:        []string{"../../../deploy/crds"},
		ControlPlaneStartTimeout: time.Second * 30,
		ControlPlaneStopTimeout:  time.Second * 3,
	}
	cfg, err := testenv.Start()
	if err != nil {
		t.Fatal("Failed to start test environment", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	t.Cleanup(func() {
		cancel()
		t.Log("Stopping test environment")
		err := testenv.Stop()
		if err != nil {
			t.Log("Failed to stop test environment", err)
		}
	})
	t.Log("Creating new controller manager")
	mgr, err := manager.NewFromConfig(cfg, manager.Options{
		ShutdownTimeout: time.Second * 3,
		DisableCache:    true,
		WebhookPort:     0,
		MetricsPort:     0,
		ProbePort:       0,
	})
	if err != nil {
		t.Fatal("Failed to create manager:", err)
	}
	t.Log("Creating new database")
	db, err := New(mgr, "default")
	if err != nil {
		t.Fatal("Failed to create database:", err)
	}
	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			t.Log("Failed to start manager:", err)
		}
	}()
	return db, NewGraphStore(mgr.GetClient(), "default")
}

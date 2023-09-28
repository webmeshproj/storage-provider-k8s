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

package main

import (
	"context"
	"flag"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	corescheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/webmeshproj/storage-provider-k8s/pkg/provider"
	"github.com/webmeshproj/storage-provider-k8s/pkg/version"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(corescheme.AddToScheme(scheme))
}

func main() {
	var opts provider.Options
	logopts := zap.Options{Development: true}
	flag.StringVar(&opts.MetricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&opts.ProbeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&opts.Namespace, "namespace", os.Getenv("NAMESPACE"), "The namespace in which to operate. Defaults to the namespace of the controller object or the NAMESPACE environment variable.")
	flag.DurationVar(&opts.LeaderElectionLeaseDuration, "leader-lease-duration", 10*time.Second, "The duration that non-leader candidates will wait to force acquire leadership. This is measured against time of last observed ack.")
	flag.DurationVar(&opts.LeaderElectionRenewDeadline, "leader-renew-deadline", 10*time.Second, "The duration that the acting leader will retry refreshing leadership before giving up.")
	flag.DurationVar(&opts.LeaderElectionRetryPeriod, "leader-retry-period", 2*time.Second, "The duration the LeaderElector clients should wait between tries of actions.")
	flag.DurationVar(&opts.ShutdownTimeout, "graceful-shutdown-timeout", time.Second*10, "The duration to wait for the controller to shutdown gracefully. If 0, the controller will not wait for graceful shutdown.")
	logopts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&logopts)))

	setupLog.Info("Starting Webmesh Storage Provider for Kubernetes",
		"version", version.Version,
		"git-commit", version.Commit,
		"build-date", version.BuildDate,
	)

	provider, err := provider.New(opts)
	if err != nil {
		setupLog.Error(err, "Failed to create storage provider")
		os.Exit(1)
	}

	if err := provider.Start(context.Background()); err != nil {
		setupLog.Error(err, "Failed to start storage provider")
		os.Exit(1)
	}
}

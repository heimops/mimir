package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	mimirio "mimir.io/operator/api/v1alpha1"
	"mimir.io/operator/internal/controller"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(mimirio.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var probeAddr string
	var webhookCertDir string
	var defaultImage string
	var enableLeaderElection bool
	var enableWebhook bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "Metrics endpoint address.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "Health probe endpoint address.")
	flag.StringVar(&webhookCertDir, "webhook-cert-dir", "/tmp/k8s-webhook-server/serving-certs", "Directory with tls.crt and tls.key for the webhook server.")
	flag.StringVar(&defaultImage, "default-mimir-image", "mimir:latest", "Default mimir app image used in CronJobs.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", true, "Enable leader election for high availability.")
	flag.BoolVar(&enableWebhook, "enable-webhook", true, "Enable the validating webhook (requires TLS certs in --webhook-cert-dir).")

	opts := zap.Options{Development: false}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	logger := ctrl.Log.WithName("setup")

	mgrOpts := ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "mimir.io",
	}
	if enableWebhook {
		mgrOpts.WebhookServer = webhook.NewServer(webhook.Options{
			Port:    9443,
			CertDir: webhookCertDir,
		})
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOpts)
	if err != nil {
		logger.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controller.BackupScheduleReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		DefaultImage: defaultImage,
	}).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to create controller")
		os.Exit(1)
	}

	if enableWebhook {
		if err = (&mimirio.BackupSchedule{}).SetupWebhookWithManager(mgr); err != nil {
			logger.Error(err, "unable to create webhook")
			os.Exit(1)
		}
		logger.Info("Validating webhook registered")
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	logger.Info("starting manager", "image", defaultImage, "webhook", enableWebhook)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error(err, "problem running manager")
		os.Exit(1)
	}
}

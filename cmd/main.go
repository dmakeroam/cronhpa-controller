package main

import (
	"flag"
	"os"

	"go.uber.org/fx"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	v1 "github.com/dmakeroam/cronhpa-controller/api/v1"
	"github.com/dmakeroam/cronhpa-controller/internal/controller"
	"github.com/dmakeroam/cronhpa-controller/internal/cron"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	app := fx.New(
		fx.Provide(
			func() (ctrl.Manager, error) {
				return ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
					Scheme:                 scheme,
					Metrics:                server.Options{BindAddress: metricsAddr},
					HealthProbeBindAddress: probeAddr,
					LeaderElection:         enableLeaderElection,
					LeaderElectionID:       "cronhpa.dmakeroam.com",
				})
			},
			func(mgr ctrl.Manager) client.Client { return mgr.GetClient() },
			func(mgr ctrl.Manager) *runtime.Scheme { return mgr.GetScheme() },
			func(c client.Client, s *runtime.Scheme) *cron.CronManager {
				return cron.NewCronManager(c, s)
			},
			func(c client.Client, s *runtime.Scheme, cm *cron.CronManager) *controller.CronHorizontalPodAutoscalerReconciler {
				return &controller.CronHorizontalPodAutoscalerReconciler{
					Client:      c,
					Scheme:      s,
					CronManager: cm,
				}
			},
		),
		fx.Invoke(func(mgr ctrl.Manager) error {
			if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
				return err
			}
			if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
				return err
			}
			return nil
		}),
		fx.Invoke(func(mgr ctrl.Manager, rec *controller.CronHorizontalPodAutoscalerReconciler) error {
			return rec.SetupWithManager(mgr)
		}),
		fx.Invoke(func(mgr ctrl.Manager) {
			if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
				setupLog.Error(err, "problem running manager")
				os.Exit(1)
			}
		}),
	)
	if err := app.Err(); err != nil {
		setupLog.Error(err, "failed to start application")
		os.Exit(1)
	}
	app.Run()
}

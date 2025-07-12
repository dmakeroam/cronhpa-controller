package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/dmakeroam/cronhpa-controller/api/v1"
	"github.com/dmakeroam/cronhpa-controller/internal/cron"
)

const finalizerName = "cronhpa.autoscaling.dmakeroam.com/finalizer"

// CronHorizontalPodAutoscalerReconciler reconciles a CronHorizontalPodAutoscaler object
type CronHorizontalPodAutoscalerReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	CronManager *cron.CronManager
}

// +kubebuilder:rbac:groups=autoscaling.dmakeroam.com,resources=cronhorizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling.dmakeroam.com,resources=cronhorizontalpodautoscalers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=autoscaling.dmakeroam.com,resources=cronhorizontalpodautoscalers/finalizers,verbs=update
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;update;patch
func (r *CronHorizontalPodAutoscalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling CronHorizontalPodAutoscaler", "name", req.Name, "namespace", req.Namespace)

	cr := &v1.CronHorizontalPodAutoscaler{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "unable to fetch CronHorizontalPodAutoscaler")
		} else {
			logger.Info("CronHorizontalPodAutoscaler not found, ignoring")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion
	if cr.DeletionTimestamp != nil {
		if controllerutil.ContainsFinalizer(cr, finalizerName) {
			logger.Info("Cleaning up jobs for deletion", "uid", cr.UID)
			r.CronManager.RemoveJobs(cr.UID)
			controllerutil.RemoveFinalizer(cr, finalizerName)
			if err := r.Update(ctx, cr); err != nil {
				logger.Error(err, "failed to remove finalizer")
				return ctrl.Result{}, err
			}
			logger.Info("Finalizer removed successfully")
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(cr, finalizerName) {
		logger.Info("Adding finalizer")
		controllerutil.AddFinalizer(cr, finalizerName)
		if err := r.Update(ctx, cr); err != nil {
			logger.Error(err, "failed to add finalizer")
			return ctrl.Result{}, err
		}
		logger.Info("Finalizer added successfully")
	}

	// Register jobs
	logger.Info("Registering jobs")
	if err := r.CronManager.RegisterJobs(ctx, cr); err != nil {
		logger.Error(err, "failed to register jobs")
		return ctrl.Result{Requeue: true}, err
	}
	logger.Info("Jobs registered successfully")

	return ctrl.Result{}, nil
}

func (r *CronHorizontalPodAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.CronHorizontalPodAutoscaler{}).
		Complete(r)
}

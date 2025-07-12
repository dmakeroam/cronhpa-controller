package cron

import (
	"context"
	"fmt"
	"sync"
	"time"

	v1 "github.com/dmakeroam/cronhpa-controller/api/v1"
	"github.com/go-co-op/gocron"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type CronManager struct {
	schedulers map[string]*gocron.Scheduler
	jobs       map[types.UID]map[string]string // UID -> jobName -> timezone
	mu         sync.Mutex
	client     client.Client
	scheme     *runtime.Scheme
}

func NewCronManager(c client.Client, s *runtime.Scheme) *CronManager {
	return &CronManager{
		schedulers: make(map[string]*gocron.Scheduler),
		jobs:       make(map[types.UID]map[string]string),
		client:     c,
		scheme:     s,
	}
}

func (m *CronManager) RegisterJobs(ctx context.Context, cr *v1.CronHorizontalPodAutoscaler) error {
	logger := log.FromContext(ctx)

	m.mu.Lock()
	defer m.mu.Unlock()

	uid := cr.UID

	// Remove existing jobs for this CR
	logger.Info("Removing existing jobs", "uid", uid)
	m.removeJobsLocked(uid)

	// Always re-initialize the job map after removal to ensure it's not nil
	m.jobs[uid] = make(map[string]string)

	for _, job := range cr.Spec.Jobs {
		if err := m.registerJob(cr, job); err != nil {
			logger.Error(err, "failed to register job", "job", job.Name)
			return err
		}
		logger.Info("Job registered", "job", job.Name, "schedule", job.Schedule, "timezone", job.Timezone)
	}
	return nil
}

func (m *CronManager) registerJob(cr *v1.CronHorizontalPodAutoscaler, job v1.Job) error {
	tz := job.Timezone
	if tz == "" {
		tz = "Asia/Bangkok"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return fmt.Errorf("invalid timezone %s: %v", tz, err)
	}

	sched, ok := m.schedulers[tz]
	if !ok {
		sched = gocron.NewScheduler(loc)
		sched.StartAsync()
		m.schedulers[tz] = sched
	}

	tag := fmt.Sprintf("%s-%s", cr.UID, job.Name)
	ns := cr.Namespace
	ref := cr.Spec.ScaleTargetRef
	reps := job.MinReplicas
	runOnce := job.RunOnce

	_, err = sched.Cron(job.Schedule).Tag(tag).Do(func() {
		jobCtx := context.Background()
		logger := log.FromContext(jobCtx)
		logger.Info("Executing scheduled job", "cr", cr.Name, "job", job.Name, "target", ref.Name, "minReplicas", reps)

		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return m.updateResource(jobCtx, ns, ref, reps)
		})
		if err != nil {
			logger.Error(err, "Failed to update resource", "cr", cr.Name, "job", job.Name)
			return
		}
		logger.Info("Resource updated successfully", "cr", cr.Name, "job", job.Name)

		if runOnce {
			logger.Info("Removing run-once job", "tag", tag)
			sched.RemoveByTag(tag)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to schedule job %s: %v", job.Name, err)
	}

	m.jobs[cr.UID][job.Name] = tz
	return nil
}

func (m *CronManager) removeJobsLocked(uid types.UID) {
	if jobMap, ok := m.jobs[uid]; ok {
		for jobName, tz := range jobMap {
			tag := fmt.Sprintf("%s-%s", uid, jobName)
			if sched, ok := m.schedulers[tz]; ok {
				sched.RemoveByTag(tag)
			}
		}
		delete(m.jobs, uid)
	}
}

func (m *CronManager) RemoveJobs(uid types.UID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	log.Log.Info("Removing jobs", "uid", uid) // Note: Using global log since no ctx here; consider passing ctx if needed
	m.removeJobsLocked(uid)
}

func (m *CronManager) updateResource(ctx context.Context, ns string, ref autoscalingv2.CrossVersionObjectReference, replicas int32) error {
	logger := log.FromContext(ctx)

	logger.Info("Updating resource", "kind", ref.Kind, "name", ref.Name, "namespace", ns, "replicas", replicas)

	if ref.Kind == "HorizontalPodAutoscaler" {
		hpa := &autoscalingv2.HorizontalPodAutoscaler{}
		if err := m.client.Get(ctx, types.NamespacedName{Namespace: ns, Name: ref.Name}, hpa); err != nil {
			return err
		}
		if hpa.Spec.MinReplicas != nil && *hpa.Spec.MinReplicas == replicas {
			logger.Info("No update needed, minReplicas already matches")
			return nil
		}
		hpa.Spec.MinReplicas = &replicas
		if err := m.client.Update(ctx, hpa); err != nil {
			return err
		}
		logger.Info("HPA minReplicas updated successfully")
		return nil
	}

	// For Deployment or StatefulSet
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(ref.APIVersion)
	obj.SetKind(ref.Kind)
	if err := m.client.Get(ctx, types.NamespacedName{Namespace: ns, Name: ref.Name}, obj); err != nil {
		return err
	}
	currentReplicas, found, err := unstructured.NestedInt64(obj.Object, "spec", "replicas")
	if err != nil {
		return err
	}
	if found && currentReplicas == int64(replicas) {
		logger.Info("No update needed, replicas already matches")
		return nil
	}
	if err := unstructured.SetNestedField(obj.Object, int64(replicas), "spec", "replicas"); err != nil {
		return err
	}
	if err := m.client.Update(ctx, obj); err != nil {
		return err
	}
	logger.Info("Resource replicas updated successfully")
	return nil
}

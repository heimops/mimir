package controller

import (
	"context"
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mimirio "mimir.io/operator/api/v1alpha1"
)

type BackupScheduleReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	DefaultImage string
}

func (r *BackupScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var bs mimirio.BackupSchedule
	if err := r.Get(ctx, req.NamespacedName, &bs); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	reconcileErr := r.sync(ctx, &bs)

	// Status update is best-effort: log the error but don't shadow the sync error.
	if err := r.updateStatus(ctx, &bs, reconcileErr); err != nil {
		log.FromContext(ctx).Error(err, "Failed to update status")
	}

	return ctrl.Result{}, reconcileErr
}

// sync reconciles the CronJob and mutates bs.Status.CronJob on success.
func (r *BackupScheduleReconciler) sync(ctx context.Context, bs *mimirio.BackupSchedule) error {
	logger := log.FromContext(ctx)

	desired := r.buildCronJob(bs)
	if err := controllerutil.SetControllerReference(bs, desired, r.Scheme); err != nil {
		return err
	}

	var existing batchv1.CronJob
	err := r.Get(ctx, client.ObjectKey{Name: desired.Name, Namespace: desired.Namespace}, &existing)
	if apierrors.IsNotFound(err) {
		if err := r.Create(ctx, desired); err != nil {
			return err
		}
		logger.Info("Created CronJob", "name", desired.Name)
	} else if err != nil {
		return err
	} else {
		existing.Spec = desired.Spec
		if err := r.Update(ctx, &existing); err != nil {
			return err
		}
		logger.Info("Updated CronJob", "name", existing.Name)
	}

	bs.Status.CronJob = desired.Name
	return nil
}

// updateStatus sets the Ready condition and persists the status subresource.
func (r *BackupScheduleReconciler) updateStatus(ctx context.Context, bs *mimirio.BackupSchedule, syncErr error) error {
	cond := metav1.Condition{
		Type:               mimirio.ConditionReady,
		ObservedGeneration: bs.Generation,
	}

	if syncErr != nil {
		cond.Status = metav1.ConditionFalse
		cond.Reason = mimirio.ReasonSyncFailed
		cond.Message = syncErr.Error()
	} else {
		cond.Status = metav1.ConditionTrue
		cond.Reason = mimirio.ReasonCronJobSynced
		cond.Message = fmt.Sprintf("CronJob %s synced successfully", bs.Status.CronJob)
	}

	apimeta.SetStatusCondition(&bs.Status.Conditions, cond)

	if err := r.Status().Update(ctx, bs); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (r *BackupScheduleReconciler) buildCronJob(bs *mimirio.BackupSchedule) *batchv1.CronJob {
	image := bs.Spec.Image
	if image == "" {
		image = r.DefaultImage
	}

	env := []corev1.EnvVar{
		{Name: "MIMIR_NAMESPACES", Value: bs.Spec.Namespace},
		{Name: "MIMIR_STORAGE_BACKEND", Value: bs.Spec.StorageBackend},
	}
	if len(bs.Spec.PVCs) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "MIMIR_PVCS",
			Value: strings.Join(bs.Spec.PVCs, ","),
		})
	}

	var envFrom []corev1.EnvFromSource
	if bs.Spec.StorageSecret != "" {
		envFrom = append(envFrom, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: bs.Spec.StorageSecret},
			},
		})
	}

	forbid := batchv1.ForbidConcurrent

	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cronJobName(bs.Name),
			Namespace: bs.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name": "mimir",
				"mimir/backup-schedule":  bs.Name,
			},
		},
		Spec: batchv1.CronJobSpec{
			Schedule:          bs.Spec.Schedule,
			ConcurrencyPolicy: forbid,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							ServiceAccountName: "mimir",
							RestartPolicy:      corev1.RestartPolicyOnFailure,
							Containers: []corev1.Container{
								{
									Name:    "mimir",
									Image:   image,
									Args:    []string{"backup"},
									Env:     env,
									EnvFrom: envFrom,
								},
							},
						},
					},
				},
			},
		},
	}
}

func cronJobName(crName string) string {
	name := fmt.Sprintf("mimir-%s", crName)
	if len(name) > 52 {
		return name[:52]
	}
	return name
}

func (r *BackupScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mimirio.BackupSchedule{}).
		Owns(&batchv1.CronJob{}).
		Complete(r)
}

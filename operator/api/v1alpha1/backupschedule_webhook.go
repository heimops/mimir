package v1alpha1

import (
	"fmt"

	"github.com/robfig/cron/v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupWebhookWithManager registers the validating webhook with the manager.
func (r *BackupSchedule) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(r).Complete()
}

// +kubebuilder:webhook:path=/validate-mimir-io-v1alpha1-backupschedule,mutating=false,failurePolicy=fail,sideEffects=None,groups=mimir.io,resources=backupschedules,verbs=create;update,versions=v1alpha1,name=vbackupschedule.mimir.io,admissionReviewVersions=v1

var _ admission.Validator = &BackupSchedule{}

func (r *BackupSchedule) ValidateCreate() (admission.Warnings, error) {
	return r.validate()
}

func (r *BackupSchedule) ValidateUpdate(_ runtime.Object) (admission.Warnings, error) {
	return r.validate()
}

func (r *BackupSchedule) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

func (r *BackupSchedule) validate() (admission.Warnings, error) {
	var errs field.ErrorList

	if _, err := cron.ParseStandard(r.Spec.Schedule); err != nil {
		errs = append(errs, field.Invalid(
			field.NewPath("spec").Child("schedule"),
			r.Spec.Schedule,
			fmt.Sprintf("invalid cron expression: %v", err),
		))
	}

	if len(errs) > 0 {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: GroupVersion.Group, Kind: "BackupSchedule"},
			r.Name,
			errs,
		)
	}
	return nil, nil
}

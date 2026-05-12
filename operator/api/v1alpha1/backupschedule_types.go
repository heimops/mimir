package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Condition type and reason constants.
const (
	ConditionReady = "Ready"

	ReasonCronJobSynced = "CronJobSynced"
	ReasonSyncFailed    = "SyncFailed"
)

// BackupScheduleSpec defines the desired state.
type BackupScheduleSpec struct {
	// Namespace whose PVCs will be backed up.
	Namespace string `json:"namespace"`

	// Schedule is a cron expression (e.g. "0 2 * * *").
	Schedule string `json:"schedule"`

	// StorageBackend selects the destination: s3, azure, or gcs.
	// +kubebuilder:validation:Enum=s3;azure;gcs
	StorageBackend string `json:"storageBackend"`

	// StorageSecret is the name of a Secret in this namespace whose keys are
	// MIMIR_* env vars with the storage credentials.
	// +optional
	StorageSecret string `json:"storageSecret,omitempty"`

	// PVCs lists specific PVC names to back up. Empty means all PVCs.
	// +optional
	PVCs []string `json:"pvcs,omitempty"`

	// Image overrides the mimir app container image for this schedule.
	// +optional
	Image string `json:"image,omitempty"`
}

// BackupScheduleStatus holds observed state.
type BackupScheduleStatus struct {
	// CronJob is the name of the managed CronJob.
	// +optional
	CronJob string `json:"cronJob,omitempty"`

	// Conditions represent the latest observations of the BackupSchedule.
	// The Ready condition is True when the CronJob is in sync with the spec.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.spec.namespace`
// +kubebuilder:printcolumn:name="Schedule",type=string,JSONPath=`.spec.schedule`
// +kubebuilder:printcolumn:name="Backend",type=string,JSONPath=`.spec.storageBackend`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// BackupSchedule is the Schema for the backupschedules API.
type BackupSchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupScheduleSpec   `json:"spec,omitempty"`
	Status BackupScheduleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BackupScheduleList contains a list of BackupSchedule.
type BackupScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupSchedule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackupSchedule{}, &BackupScheduleList{})
}

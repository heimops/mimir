package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newBS(schedule string) *BackupSchedule {
	return &BackupSchedule{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: BackupScheduleSpec{
			Namespace:      "production",
			Schedule:       schedule,
			StorageBackend: "s3",
		},
	}
}

func TestValidateCreate(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		wantErr  bool
	}{
		{"daily at 2am", "0 2 * * *", false},
		{"every minute", "* * * * *", false},
		{"every hour", "0 * * * *", false},
		{"first of month", "0 0 1 * *", false},
		{"empty string", "", true},
		{"plain text", "every day", true},
		{"too few fields", "0 2 * *", true},
		{"too many fields", "0 0 2 * * *", true},
		{"invalid range", "60 * * * *", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newBS(tt.schedule).ValidateCreate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreate(%q) error = %v, wantErr %v", tt.schedule, err, tt.wantErr)
			}
		})
	}
}

func TestValidateUpdate(t *testing.T) {
	_, err := newBS("0 3 * * *").ValidateUpdate(nil)
	if err != nil {
		t.Errorf("ValidateUpdate() unexpected error: %v", err)
	}

	_, err = newBS("bad cron").ValidateUpdate(nil)
	if err == nil {
		t.Error("ValidateUpdate() expected error for bad cron, got nil")
	}
}

func TestValidateDelete(t *testing.T) {
	_, err := newBS("0 2 * * *").ValidateDelete()
	if err != nil {
		t.Errorf("ValidateDelete() unexpected error: %v", err)
	}
}

func TestErrorMessageContainsCronExpression(t *testing.T) {
	badSchedule := "not-a-cron"
	_, err := newBS(badSchedule).ValidateCreate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if len(msg) == 0 {
		t.Error("expected non-empty error message")
	}
	// The field path must identify spec.schedule
	for _, substr := range []string{"spec.schedule", "invalid cron"} {
		if !contains(msg, substr) {
			t.Errorf("error message %q missing expected substring %q", msg, substr)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsHelper(s, sub))
}

func containsHelper(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

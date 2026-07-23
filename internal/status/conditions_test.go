package status

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
)

func TestSetCondition_NewCondition(t *testing.T) {
	s := &platformv1alpha1.PlatformApplicationStatus{}

	SetCondition(s, platformv1alpha1.ConditionReady, metav1.ConditionTrue, "Available", "All replicas ready")

	if len(s.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(s.Conditions))
	}

	c := s.Conditions[0]
	if c.Type != platformv1alpha1.ConditionReady {
		t.Errorf("expected type Ready, got %s", c.Type)
	}
	if c.Status != metav1.ConditionTrue {
		t.Errorf("expected status True, got %s", c.Status)
	}
	if c.Reason != "Available" {
		t.Errorf("expected reason Available, got %s", c.Reason)
	}
	if c.Message != "All replicas ready" {
		t.Errorf("expected message 'All replicas ready', got %s", c.Message)
	}
	if c.LastTransitionTime.IsZero() {
		t.Error("expected LastTransitionTime to be set")
	}
}

func TestSetCondition_UpdateExisting(t *testing.T) {
	s := &platformv1alpha1.PlatformApplicationStatus{}

	// Set initial condition.
	SetCondition(s, platformv1alpha1.ConditionReady, metav1.ConditionFalse, "Pending", "Waiting for pods")
	initialTime := s.Conditions[0].LastTransitionTime

	// Update with same values — should not change transition time.
	SetCondition(s, platformv1alpha1.ConditionReady, metav1.ConditionFalse, "Pending", "Waiting for pods")
	if !s.Conditions[0].LastTransitionTime.Equal(&initialTime) {
		t.Error("expected LastTransitionTime to remain unchanged for same values")
	}

	// Update with different status — should update transition time.
	SetCondition(s, platformv1alpha1.ConditionReady, metav1.ConditionTrue, "Available", "All replicas ready")
	if len(s.Conditions) != 1 {
		t.Errorf("expected 1 condition (updated in place), got %d", len(s.Conditions))
	}
	if s.Conditions[0].Status != metav1.ConditionTrue {
		t.Errorf("expected status True after update, got %s", s.Conditions[0].Status)
	}
}

func TestSetCondition_MultipleConditions(t *testing.T) {
	s := &platformv1alpha1.PlatformApplicationStatus{}

	SetReady(s, metav1.ConditionTrue, "Available", "Ready")
	SetProgressing(s, metav1.ConditionFalse, "Deployed", "Done")
	SetDegraded(s, metav1.ConditionFalse, "Healthy", "OK")
	SetConfigurationValid(s, metav1.ConditionTrue, "Valid", "Spec is valid")

	if len(s.Conditions) != 4 {
		t.Fatalf("expected 4 conditions, got %d", len(s.Conditions))
	}

	// Verify each condition exists.
	condMap := make(map[string]metav1.Condition)
	for _, c := range s.Conditions {
		condMap[c.Type] = c
	}

	if condMap[platformv1alpha1.ConditionReady].Status != metav1.ConditionTrue {
		t.Error("expected Ready=True")
	}
	if condMap[platformv1alpha1.ConditionProgressing].Status != metav1.ConditionFalse {
		t.Error("expected Progressing=False")
	}
	if condMap[platformv1alpha1.ConditionDegraded].Status != metav1.ConditionFalse {
		t.Error("expected Degraded=False")
	}
	if condMap[platformv1alpha1.ConditionConfigurationValid].Status != metav1.ConditionTrue {
		t.Error("expected ConfigurationValid=True")
	}
}

func TestIsReady(t *testing.T) {
	tests := []struct {
		name     string
		status   *platformv1alpha1.PlatformApplicationStatus
		expected bool
	}{
		{
			name:     "no conditions",
			status:   &platformv1alpha1.PlatformApplicationStatus{},
			expected: false,
		},
		{
			name: "ready true",
			status: &platformv1alpha1.PlatformApplicationStatus{
				Conditions: []metav1.Condition{
					{Type: platformv1alpha1.ConditionReady, Status: metav1.ConditionTrue},
				},
			},
			expected: true,
		},
		{
			name: "ready false",
			status: &platformv1alpha1.PlatformApplicationStatus{
				Conditions: []metav1.Condition{
					{Type: platformv1alpha1.ConditionReady, Status: metav1.ConditionFalse},
				},
			},
			expected: false,
		},
		{
			name: "other conditions but no ready",
			status: &platformv1alpha1.PlatformApplicationStatus{
				Conditions: []metav1.Condition{
					{Type: platformv1alpha1.ConditionProgressing, Status: metav1.ConditionTrue},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsReady(tt.status)
			if result != tt.expected {
				t.Errorf("expected IsReady=%v, got %v", tt.expected, result)
			}
		})
	}
}

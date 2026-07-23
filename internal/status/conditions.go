package status

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
)

// SetCondition sets or updates a condition on the PlatformApplication status.
// It only updates if the status, reason, or message has changed to avoid
// unnecessary API writes.
func SetCondition(status *platformv1alpha1.PlatformApplicationStatus, conditionType string, condStatus metav1.ConditionStatus, reason, message string) {
	now := metav1.NewTime(time.Now())

	for i, c := range status.Conditions {
		if c.Type == conditionType {
			if c.Status != condStatus || c.Reason != reason || c.Message != message {
				status.Conditions[i].Status = condStatus
				status.Conditions[i].Reason = reason
				status.Conditions[i].Message = message
				status.Conditions[i].LastTransitionTime = now
			}
			return
		}
	}

	status.Conditions = append(status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             condStatus,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
	})
}

// SetReady sets the Ready condition.
func SetReady(status *platformv1alpha1.PlatformApplicationStatus, condStatus metav1.ConditionStatus, reason, message string) {
	SetCondition(status, platformv1alpha1.ConditionReady, condStatus, reason, message)
}

// SetProgressing sets the Progressing condition.
func SetProgressing(status *platformv1alpha1.PlatformApplicationStatus, condStatus metav1.ConditionStatus, reason, message string) {
	SetCondition(status, platformv1alpha1.ConditionProgressing, condStatus, reason, message)
}

// SetDegraded sets the Degraded condition.
func SetDegraded(status *platformv1alpha1.PlatformApplicationStatus, condStatus metav1.ConditionStatus, reason, message string) {
	SetCondition(status, platformv1alpha1.ConditionDegraded, condStatus, reason, message)
}

// SetConfigurationValid sets the ConfigurationValid condition.
func SetConfigurationValid(status *platformv1alpha1.PlatformApplicationStatus, condStatus metav1.ConditionStatus, reason, message string) {
	SetCondition(status, platformv1alpha1.ConditionConfigurationValid, condStatus, reason, message)
}

// IsReady returns true if the Ready condition is True.
func IsReady(status *platformv1alpha1.PlatformApplicationStatus) bool {
	for _, c := range status.Conditions {
		if c.Type == platformv1alpha1.ConditionReady {
			return c.Status == metav1.ConditionTrue
		}
	}
	return false
}

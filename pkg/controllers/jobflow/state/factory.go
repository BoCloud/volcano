package state

import (
	"volcano.sh/apis/pkg/apis/flow/v1alpha1"
)

type State interface {
	// Execute executes the actions based on current state.
	Execute(action v1alpha1.Action) error
}

// UpdateQueueStatusFn updates the queue status.
type UpdateJobFlowStatusFn func(status *v1alpha1.JobFlowStatus, allJobList int)

type JobFlowActionFn func(jobflow *v1alpha1.JobFlow, fn UpdateJobFlowStatusFn) error

var (
	// SyncJobFlow will sync queue status.
	SyncJobFlow JobFlowActionFn
)

// NewState gets the state from queue status.
func NewState(jobFlow *v1alpha1.JobFlow) State {
	switch jobFlow.Status.State.Phase {
	case "", v1alpha1.Pending:
		return &pendingState{jobFlow: jobFlow}
	case v1alpha1.Running:
		return &runningState{jobFlow: jobFlow}
	case v1alpha1.Succeed:
		return &succeedState{jobFlow: jobFlow}
	case v1alpha1.Terminating:
		return &terminatingState{jobFlow: jobFlow}
	case v1alpha1.Failed:
		return &failedState{jobFlow: jobFlow}
	}

	return nil
}

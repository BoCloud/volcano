package state

import (
	jobflowv1alpha1 "volcano.sh/apis/pkg/apis/flow/v1alpha1"
)

type pendingState struct {
	jobFlow *jobflowv1alpha1.JobFlow
}

func (p *pendingState) Execute(action jobflowv1alpha1.Action) error {
	switch action {
	case jobflowv1alpha1.SyncJobFlowAction:
		return SyncJobFlow(p.jobFlow, func(status *jobflowv1alpha1.JobFlowStatus, allJobList int) {
			if len(status.RunningJobs) > 0 || len(status.CompletedJobs) > 0 {
				status.State.Phase = jobflowv1alpha1.Running
			} else if len(status.FailedJobs) > 0 {
				status.State.Phase = jobflowv1alpha1.Failed
			} else {
				status.State.Phase = jobflowv1alpha1.Pending
			}
		})
	}
	return nil
}

package state

import "volcano.sh/apis/pkg/apis/flow/v1alpha1"

type runningState struct {
	jobFlow *v1alpha1.JobFlow
}

func (p *runningState) Execute(action v1alpha1.Action) error {
	switch action {
	case v1alpha1.SyncJobFlowAction:
		return SyncJobFlow(p.jobFlow, func(status *v1alpha1.JobFlowStatus, allJobList int) {
			if len(status.CompletedJobs) == allJobList {
				status.State.Phase = v1alpha1.Succeed
			}
		})
	}
	return nil
}

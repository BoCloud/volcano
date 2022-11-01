package state

import "volcano.sh/apis/pkg/apis/flow/v1alpha1"

type succeedState struct {
	jobFlow *v1alpha1.JobFlow
}

func (p *succeedState) Execute(action v1alpha1.Action) error {
	switch action {
	case v1alpha1.SyncJobFlowAction:
		return SyncJobFlow(p.jobFlow, func(status *v1alpha1.JobFlowStatus, allJobList int) {})
	}
	return nil
}

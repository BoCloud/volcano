package state

import "volcano.sh/apis/pkg/apis/flow/v1alpha1"

type failedState struct {
	jobFlow *v1alpha1.JobFlow
}

func (p *failedState) Execute(action v1alpha1.Action) error {
	switch action {
	case v1alpha1.SyncJobFlowAction:
		return nil
	}
	return nil
}

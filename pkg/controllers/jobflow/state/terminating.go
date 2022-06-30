package state

import "volcano.sh/apis/pkg/apis/flow/v1alpha1"

type terminatingState struct {
	jobFlow *v1alpha1.JobFlow
}

func (p *terminatingState) Execute(action v1alpha1.Action) error {
	return nil
}

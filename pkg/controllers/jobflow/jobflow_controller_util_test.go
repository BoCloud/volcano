package jobflow

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

func TestGetJobNameFunc(t *testing.T) {
	type args struct {
		jobFlowName     string
		jobTemplateName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "GetJobName success case",
			args: args{
				jobFlowName:     "jobFlowA",
				jobTemplateName: "jobTemplateA",
			},
			want: "jobFlowA-jobTemplateA",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getJobName(tt.args.jobFlowName, tt.args.jobTemplateName); got != tt.want {
				t.Errorf("getJobName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetConnectionOfJobAndJobTemplate(t *testing.T) {
	type args struct {
		namespace string
		name      string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "TestGetConnectionOfJobAndJobTemplate",
			args: args{
				namespace: "default",
				name:      "flow",
			},
			want: "default.flow",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetConnectionOfJobAndJobTemplate(tt.args.namespace, tt.args.name); got != tt.want {
				t.Errorf("GetConnectionOfJobAndJobTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetJobFlowNameByJob(t *testing.T) {
	type args struct {
		job *batch.Job
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "TestGetConnectionOfJobAndJobTemplate",
			args: args{
				job: &batch.Job{
					TypeMeta: v1.TypeMeta{},
					ObjectMeta: v1.ObjectMeta{
						OwnerReferences: []v1.OwnerReference{
							{
								APIVersion:         "flow.volcano.sh/v1alpha1",
								Kind:               JobFlow,
								Name:               "jobflowtest",
								UID:                "",
								Controller:         nil,
								BlockOwnerDeletion: nil,
							},
						},
					},
					Spec:   batch.JobSpec{},
					Status: batch.JobStatus{},
				},
			},
			want: "jobflowtest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getJobFlowNameByJob(tt.args.job); got != tt.want {
				t.Errorf("GetConnectionOfJobAndJobTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

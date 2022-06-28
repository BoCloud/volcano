package jobtemplate

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	jobflowv1alpha1 "volcano.sh/apis/pkg/apis/flow/v1alpha1"
)

func TestAddJobTemplateFunc(t *testing.T) {
	namespace := "test"

	testCases := []struct {
		Name        string
		jobTemplate *jobflowv1alpha1.JobTemplate
		ExpectValue int
	}{
		{
			Name: "AddJobTemplate Success",
			jobTemplate: &jobflowv1alpha1.JobTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "jobtemplate1",
					Namespace: namespace,
				},
			},
			ExpectValue: 1,
		},
	}
	for i, testcase := range testCases {
		t.Run(testcase.Name, func(t *testing.T) {
			fakeController := newFakeController()
			fakeController.addJobTemplate(testcase.jobTemplate)
			queueLen := fakeController.queue.Len()
			if testcase.ExpectValue != queueLen {
				t.Errorf("case %d (%s): expected: %v, got %v ", i, testcase.Name, testcase.ExpectValue, queueLen)
			}
		})
	}
}

func TestAddJob(t *testing.T) {
	namespace := "test"

	testCases := []struct {
		Name        string
		job         *batch.Job
		ExpectValue int
	}{
		{
			Name: "AddJob Success",
			job: &batch.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "job1",
					Namespace:   namespace,
					Annotations: map[string]string{CreateByJobTemplate: "test.jobtemplate1"},
				},
			},
			ExpectValue: 1,
		},
	}
	for i, testcase := range testCases {
		t.Run(testcase.Name, func(t *testing.T) {
			fakeController := newFakeController()
			fakeController.addJob(testcase.job)
			queueLen := fakeController.queue.Len()
			if testcase.ExpectValue != queueLen {
				t.Errorf("case %d (%s): expected: %v, got %v ", i, testcase.Name, testcase.ExpectValue, queueLen)
			}
		})
	}
}

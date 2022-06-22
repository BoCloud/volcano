package jobflow

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	jobflowv1alpha1 "volcano.sh/apis/pkg/apis/flow/v1alpha1"
	"volcano.sh/apis/pkg/client/clientset/versioned/scheme"
)

func TestAddJobFlowFunc(t *testing.T) {
	namespace := "test"

	testCases := []struct {
		Name        string
		jobFlow     *jobflowv1alpha1.JobFlow
		ExpectValue int
	}{
		{
			Name: "AddJobFlow Success",
			jobFlow: &jobflowv1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "jobflow1",
					Namespace: namespace,
				},
			},
			ExpectValue: 1,
		},
	}
	for i, testcase := range testCases {
		t.Run(testcase.Name, func(t *testing.T) {
			fakeController := newFakeController()
			fakeController.addJobFlow(testcase.jobFlow)
			len := fakeController.queue.Len()
			if testcase.ExpectValue != len {
				t.Errorf("case %d (%s): expected: %v, got %v ", i, testcase.Name, testcase.ExpectValue, len)
			}
		})
	}
}

func TestUpdateJobFlowFunc(t *testing.T) {
	namespace := "test"

	testCases := []struct {
		Name        string
		newJobFlow  *jobflowv1alpha1.JobFlow
		oldJobFlow  *jobflowv1alpha1.JobFlow
		ExpectValue int
	}{
		{
			Name: "UpdateJobFlow Success",
			newJobFlow: &jobflowv1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "jobflow1",
					Namespace: namespace,
				},
				Spec: jobflowv1alpha1.JobFlowSpec{
					Flows:           nil,
					JobRetainPolicy: jobflowv1alpha1.Delete,
				},
				Status: jobflowv1alpha1.JobFlowStatus{
					State: jobflowv1alpha1.State{
						Phase: jobflowv1alpha1.Succeed,
					},
				},
			},
			oldJobFlow: &jobflowv1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "jobflow1",
					Namespace:       namespace,
					ResourceVersion: "1223",
				},
				Spec: jobflowv1alpha1.JobFlowSpec{
					Flows:           nil,
					JobRetainPolicy: jobflowv1alpha1.Delete,
				},
				Status: jobflowv1alpha1.JobFlowStatus{
					State: jobflowv1alpha1.State{
						Phase: jobflowv1alpha1.Succeed,
					},
				},
			},
			ExpectValue: 1,
		},
	}
	for i, testcase := range testCases {
		t.Run(testcase.Name, func(t *testing.T) {
			fakeController := newFakeController()
			fakeController.updateJobFlow(testcase.oldJobFlow, testcase.newJobFlow)
			len := fakeController.queue.Len()
			if testcase.ExpectValue != len {
				t.Errorf("case %d (%s): expected: %v, got %v ", i, testcase.Name, testcase.ExpectValue, len)
			}
		})
	}
}

func TestUpdateJobFunc(t *testing.T) {
	namespace := "test"

	testCases := []struct {
		Name        string
		newJob      *batch.Job
		oldJob      *batch.Job
		ExpectValue int
	}{
		{
			Name: "UpdateJob Success",
			newJob: &batch.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "job1",
					Namespace: namespace,
				},
			},
			oldJob: &batch.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "job1",
					Namespace:       namespace,
					ResourceVersion: "1223",
					OwnerReferences: []metav1.OwnerReference{},
				},
			},
			ExpectValue: 1,
		},
	}
	jobFlow := &jobflowv1alpha1.JobFlow{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jobflow1",
			Namespace: namespace,
		},
		Spec:   jobflowv1alpha1.JobFlowSpec{},
		Status: jobflowv1alpha1.JobFlowStatus{},
	}
	for i, testcase := range testCases {
		t.Run(testcase.Name, func(t *testing.T) {
			fakeController := newFakeController()

			if err := controllerutil.SetControllerReference(jobFlow, testcase.oldJob, scheme.Scheme); err != nil {
				t.Errorf("SetControllerReference error : %s", err.Error())
			}
			if err := controllerutil.SetControllerReference(jobFlow, testcase.newJob, scheme.Scheme); err != nil {
				t.Errorf("SetControllerReference error : %s", err.Error())
			}
			fakeController.updateJob(testcase.oldJob, testcase.newJob)
			len := fakeController.queue.Len()
			if testcase.ExpectValue != len {
				t.Errorf("case %d (%s): expected: %v, got %v ", i, testcase.Name, testcase.ExpectValue, len)
			}
		})
	}
}

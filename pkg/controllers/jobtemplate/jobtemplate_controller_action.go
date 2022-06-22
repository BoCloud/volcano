package jobtemplate

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"

	"volcano.sh/apis/pkg/apis/batch/v1alpha1"
	v1alpha1flow "volcano.sh/apis/pkg/apis/flow/v1alpha1"
)

func (j *jobtemplatecontroller) syncJobTemplate(jobTemplate *v1alpha1flow.JobTemplate) error {
	// search the jobs created by JobTemplate
	selector := labels.NewSelector()
	jobList, err := j.jobLister.Jobs(jobTemplate.Namespace).List(selector)
	if err != nil {
		klog.Errorf("Failed to list jobs of JobTemplate %v/%v: %v",
			jobTemplate.Namespace, jobTemplate.Name, err)
		return err
	}

	filterJobList := make([]*v1alpha1.Job, 0)
	for _, job := range jobList {
		if job.Annotations[CreateByJobTemplate] == GetConnectionOfJobAndJobTemplate(jobTemplate.Namespace, jobTemplate.Name) {
			filterJobList = append(filterJobList, job)
		}
	}

	if len(filterJobList) == 0 {
		return nil
	}

	jobListName := make([]string, 0)
	for _, job := range filterJobList {
		jobListName = append(jobListName, job.Name)
	}
	jobTemplate.Status.JobDependsOnList = jobListName

	//update jobTemplate status
	_, err = j.vcClient.FlowV1alpha1().JobTemplates(jobTemplate.Namespace).UpdateStatus(context.Background(), jobTemplate, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Failed to update status of JobTemplate %v/%v: %v",
			jobTemplate.Namespace, jobTemplate.Name, err)
		return err
	}
	return nil
}

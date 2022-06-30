package jobtemplate

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/klog"
	v1alpha1flow "volcano.sh/apis/pkg/apis/flow/v1alpha1"
)

func (j *jobtemplatecontroller) syncJobTemplate(jobTemplate *v1alpha1flow.JobTemplate) error {
	// search the jobs created by JobTemplate
	selector := labels.NewSelector()
	r, err := labels.NewRequirement(CreateByJobTemplate, selection.Equals, []string{GetConnectionOfJobAndJobTemplate(jobTemplate.Namespace, jobTemplate.Name)})
	if err != nil {
		return err
	}
	selector = selector.Add(*r)
	jobList, err := j.jobLister.Jobs(jobTemplate.Namespace).List(selector)
	if err != nil {
		klog.Errorf("Failed to list jobs of JobTemplate %v/%v: %v",
			jobTemplate.Namespace, jobTemplate.Name, err)
		return err
	}

	if len(jobList) == 0 {
		return nil
	}

	jobListName := make([]string, 0)
	for _, job := range jobList {
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

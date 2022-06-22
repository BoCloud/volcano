package jobtemplate

import (
	"strings"

	"k8s.io/klog"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	"volcano.sh/apis/pkg/apis/flow/v1alpha1"
	"volcano.sh/volcano/pkg/controllers/apis"
)

func (j *jobtemplatecontroller) enqueue(req *apis.FlowRequest) {
	j.queue.Add(req)
}

func (j *jobtemplatecontroller) addJobTemplate(obj interface{}) {
	jobTemplate, ok := obj.(*v1alpha1.JobTemplate)
	if !ok {
		klog.Errorf("Failed to convert %v to jobTemplate", obj)
		return
	}

	req := &apis.FlowRequest{
		Namespace:       jobTemplate.Namespace,
		JobTemplateName: jobTemplate.Name,
	}

	j.enqueueJobTemplate(req)
}

func (j *jobtemplatecontroller) addJob(obj interface{}) {
	job, ok := obj.(*batch.Job)
	if !ok {
		klog.Errorf("Failed to convert %v to vcjob", obj)
		return
	}

	if job.GetAnnotations()[CreateByJobTemplate] == "" {
		return
	}

	namespaceName := strings.Split(job.GetAnnotations()[CreateByJobTemplate], ".")
	if len(namespaceName) != CreateByJobTemplateValueNum {
		return
	}
	namespace, name := namespaceName[0], namespaceName[1]

	req := &apis.FlowRequest{
		Namespace:       namespace,
		JobTemplateName: name,
	}
	j.queue.Add(req)
}

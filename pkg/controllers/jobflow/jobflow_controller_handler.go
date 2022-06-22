package jobflow

import (
	"k8s.io/klog"

	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	jobflowv1alpha1 "volcano.sh/apis/pkg/apis/flow/v1alpha1"
	"volcano.sh/apis/pkg/apis/helpers"
	"volcano.sh/volcano/pkg/controllers/apis"
)

func (j *jobflowcontroller) enqueue(req *apis.FlowRequest) {
	j.queue.Add(req)
}

func (j *jobflowcontroller) addJobFlow(obj interface{}) {
	jobFlow, ok := obj.(*jobflowv1alpha1.JobFlow)
	if !ok {
		klog.Errorf("Failed to convert %v to jobFlow", obj)
		return
	}

	req := &apis.FlowRequest{
		Namespace:   jobFlow.Namespace,
		JobFlowName: jobFlow.Name,

		Action: jobflowv1alpha1.SyncJobFlowAction,
		Event:  jobflowv1alpha1.OutOfSyncEvent,
	}

	j.enqueueJobFlow(req)
}

func (j *jobflowcontroller) updateJobFlow(oldObj, newObj interface{}) {
	oldJobFlow, ok := oldObj.(*jobflowv1alpha1.JobFlow)
	if !ok {
		klog.Errorf("Failed to convert %v to vcjob", oldJobFlow)
		return
	}

	newJobFlow, ok := newObj.(*jobflowv1alpha1.JobFlow)
	if !ok {
		klog.Errorf("Failed to convert %v to vcjob", newJobFlow)
		return
	}

	if newJobFlow.ResourceVersion == oldJobFlow.ResourceVersion {
		return
	}

	if newJobFlow.Status.State.Phase != jobflowv1alpha1.Succeed || newJobFlow.Spec.JobRetainPolicy != jobflowv1alpha1.Delete {
		return
	}

	req := &apis.FlowRequest{
		Namespace:   newJobFlow.Namespace,
		JobFlowName: newJobFlow.Name,

		Action: jobflowv1alpha1.SyncJobFlowAction,
		Event:  jobflowv1alpha1.OutOfSyncEvent,
	}

	j.enqueueJobFlow(req)
}

func (j *jobflowcontroller) updateJob(oldObj, newObj interface{}) {

	oldJob, ok := oldObj.(*batch.Job)
	if !ok {
		klog.Errorf("Failed to convert %v to vcjob", oldObj)
		return
	}

	newJob, ok := newObj.(*batch.Job)
	if !ok {
		klog.Errorf("Failed to convert %v to vcjob", newObj)
		return
	}

	// Filter out jobs that are not created from volcano jobflow
	if !isControlledBy(newJob, helpers.JobFlowKind) {
		return
	}

	if newJob.ResourceVersion == oldJob.ResourceVersion {
		return
	}

	jobFlowName := getJobFlowNameByJob(newJob)
	if jobFlowName == "" {
		return
	}

	req := &apis.FlowRequest{
		Namespace:   newJob.Namespace,
		JobFlowName: jobFlowName,
		Action:      jobflowv1alpha1.SyncJobFlowAction,
		Event:       jobflowv1alpha1.OutOfSyncEvent,
	}

	j.queue.Add(req)
}

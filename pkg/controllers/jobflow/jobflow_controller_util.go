package jobflow

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

func getJobName(jobFlowName string, jobTemplateName string) string {
	return jobFlowName + "-" + jobTemplateName
}

func GetConnectionOfJobAndJobTemplate(namespace, name string) string {
	return namespace + "." + name
}

func isControlledBy(obj metav1.Object, gvk schema.GroupVersionKind) bool {
	controllerRef := metav1.GetControllerOf(obj)
	if controllerRef == nil {
		return false
	}
	if controllerRef.APIVersion == gvk.GroupVersion().String() && controllerRef.Kind == gvk.Kind {
		return true
	}
	return false
}

func getJobFlowNameByJob(job *batch.Job) string {
	for _, owner := range job.OwnerReferences {
		if owner.Kind == JobFlow && strings.Contains(owner.APIVersion, Volcano) {
			return owner.Name
		}
	}
	return ""
}

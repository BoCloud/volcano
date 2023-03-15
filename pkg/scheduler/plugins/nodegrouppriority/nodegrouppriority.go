/*
Copyright 2023 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License")
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http: //www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nodegrouppriority

import (
	"sort"
	"strconv"
	"strings"

	"k8s.io/klog"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	"volcano.sh/volcano/pkg/scheduler/api"
	"volcano.sh/volcano/pkg/scheduler/framework"
)

// User should specify arguments in the config in this format:
//
//  actions: "reclaim, allocate, backfill, preempt"
//  tiers:
//  - plugins:
//    - name: priority
//    - name: gang
//    - name: conformance
//  - plugins:
//    - name: drf
//    - name: predicates
//    - name: proportion
//    - name: nodegrouppriority
//      enableNodeOrder: true

const (
	// PluginName indicates name of volcano scheduler plugin.
	PluginName = "nodegrouppriority"
	//NodeGroupNameKey = "volcano.sh/nodegroup-name"
	// NodeGroupNameKey the same value is grouped into the same group
	NodeGroupNameKey = "volcano.sh/nodegrouppriority"

	// NodeGroupNameKeyZero is the default volcano.sh/nodegroup value
	NodeGroupNameKeyZero = 0

	// NodeGroupWeight is the default volcano.sh/nodegroup weight
	NodeGroupWeight = 100000
)

type nodeGroupPlugin struct {
	// Arguments given for the plugin
	pluginArguments framework.Arguments
}

// New function returns prioritize plugin object.
func New(arguments framework.Arguments) framework.Plugin {
	return &nodeGroupPlugin{pluginArguments: arguments}
}

func (np *nodeGroupPlugin) Name() string {
	return PluginName
}

//nodeGroupsScore Evaluation node score
func nodeGroupsScore(priorityNodeGroups []*PriorityNodeGroup, node *api.NodeInfo) float64 {
	klog.V(4).Infof("nodeGroupsLen: %v,", len(priorityNodeGroups))
	nodeScore := 0.0
	if len(priorityNodeGroups) <= 1 {
		return nodeScore
	}
	for _, priorityNodeGroup := range priorityNodeGroups {
		for _, nodeInfo := range priorityNodeGroup.NodeInfos {
			if nodeInfo.Node.Name == node.Name {
				return float64(priorityNodeGroup.Priority * NodeGroupWeight)
			}
		}
	}
	return nodeScore
}

type PriorityNodeGroup struct {
	Priority  int
	NodeInfos []*api.NodeInfo
}

// GetPriorityNodeGroup Get the priority of the node group
func GetPriorityNodeGroup(ssn *framework.Session) []*PriorityNodeGroup {
	nodeGroups := make(map[int][]*api.NodeInfo)
	for _, node := range ssn.Nodes {
		priority, found := node.Node.Labels[NodeGroupNameKey]
		index := strings.Index(priority, "-")
		if found {
			var priorityInt int
			var err error
			if index != -1 {
				priorityInt, err = strconv.Atoi(priority[0:index])
			} else {
				priorityInt, err = strconv.Atoi(priority)
			}
			if err != nil {
				getNodes, found := nodeGroups[NodeGroupNameKeyZero]
				if found {
					getNodes = append(getNodes, node)
					nodeGroups[NodeGroupNameKeyZero] = getNodes
				} else {
					nodes := make([]*api.NodeInfo, 0)
					nodes = append(nodes, node)
					nodeGroups[NodeGroupNameKeyZero] = nodes
				}
			} else {
				getNodes, found := nodeGroups[priorityInt]
				if found {
					getNodes = append(getNodes, node)
					nodeGroups[priorityInt] = getNodes
				} else {
					nodes := make([]*api.NodeInfo, 0)
					nodes = append(nodes, node)
					nodeGroups[priorityInt] = nodes
				}
			}
		} else {
			getNodes, found := nodeGroups[NodeGroupNameKeyZero]
			if found {
				getNodes = append(getNodes, node)
				nodeGroups[NodeGroupNameKeyZero] = getNodes
			} else {
				nodes := make([]*api.NodeInfo, 0)
				nodes = append(nodes, node)
				nodeGroups[NodeGroupNameKeyZero] = nodes
			}
		}
	}

	nodeGroupsSort := make([]*PriorityNodeGroup, 0)
	for priority, nodeGroup := range nodeGroups {
		nodeGroupsSort = append(nodeGroupsSort, &PriorityNodeGroup{
			Priority:  priority,
			NodeInfos: nodeGroup,
		})
	}
	sort.Slice(nodeGroupsSort, func(i, j int) bool {
		return nodeGroupsSort[i].Priority < nodeGroupsSort[j].Priority
	})
	return nodeGroupsSort
}

func (np *nodeGroupPlugin) OnSessionOpen(ssn *framework.Session) {
	nodeGroups := GetPriorityNodeGroup(ssn)
	nodeOrderFn := func(task *api.TaskInfo, node *api.NodeInfo) (float64, error) {
		group := node.Node.Labels[NodeGroupNameKey]
		queue := task.Pod.Labels[batch.QueueNameKey]
		score := nodeGroupsScore(nodeGroups, node)
		klog.V(4).Infof("task <%s>/<%s> queue %s on node %s of nodegroup %s, score %v", task.Namespace, task.Name, queue, node.Name, group, score)
		return score, nil
	}
	ssn.AddNodeOrderFn(np.Name(), nodeOrderFn)
}

func (np *nodeGroupPlugin) OnSessionClose(ssn *framework.Session) {
}

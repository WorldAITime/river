package model

import "k8s.io/apimachinery/pkg/watch"

const (
	DEPLOYMENT = "deployments"
)

type Listener struct {
	ResourceType string `json:"type"`
	ResourceName string `json:"name"`
	Namespace    string `json:"namespace"`
	NotifyChan   chan watch.Event
}

type Metadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type Status struct {
	Replicas          int32 `json:"replicas"`
	ReadyReplicas     int32 `json:"readyReplicas"`
	AvailableReplicas int32 `json:"availableReplicas"`
}

type MiniDeployment struct {
	Metadata Metadata `json:"metadata"`
	Status   Status   `json:"status"`
}

type InformMessage struct {
	Resource  interface{} `json:"resource"`
	EventType string      `json:"eventtype"`
}

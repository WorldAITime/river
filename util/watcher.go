package util

import (
	. "async/model"
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/watch"
	v1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"sync"
)

type WatchController struct {
	listeners map[string][]*Listener

	deploymentWatch watch.Interface

	mu sync.Mutex
}

func NewWatchController(depWatch watch.Interface) *WatchController {
	return &WatchController{
		listeners:       make(map[string][]*Listener),
		deploymentWatch: depWatch,
	}
}

func (wc *WatchController) Register(listener *Listener) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	if _, ok := wc.listeners[listener.ResourceType]; !ok {
		wc.listeners[listener.ResourceType] = make([]*Listener, 0)
	}

	wc.listeners[listener.ResourceType] = append(wc.listeners[listener.ResourceType], listener)
	fmt.Printf("register resourceType: %s, resource key: %s/%s\n", listener.ResourceType, listener.ResourceName, listener.Namespace)
	return nil
}

func (wc *WatchController) UnRegister(listener *Listener) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	if _, ok := wc.listeners[listener.ResourceType]; !ok {
		fmt.Println("Unable to find related listener.")
		return errors.New("Unable to find related listener.")
	}

	for i := 0; i < len(wc.listeners[listener.ResourceType]); i++ {
		if wc.listeners[listener.ResourceType][i] == listener {
			wc.listeners[listener.ResourceType] = append(wc.listeners[listener.ResourceType][:i], wc.listeners[listener.ResourceType][i+1:]...)
			fmt.Printf("unregister resourceType: %s, resource key: %s/%s\n", listener.ResourceType, listener.ResourceName, listener.Namespace)
			return nil
		}
	}

	return errors.New("Unable to find related listener.")
}

func (wc *WatchController) Run() {
	for {
		select {
		case t := <-wc.deploymentWatch.ResultChan():
			if t.Type == watch.Modified {
				wc.InformListener(DEPLOYMENT, t)
			}
		}
	}
}

func (wc *WatchController) InformListener(resourceType string, e watch.Event) {
	dep := e.Object.(*v1beta1.Deployment)
	for i := 0; i < len(wc.listeners[resourceType]); i++ {
		listener := wc.listeners[resourceType][i]
		if listener.ResourceName == dep.Name && listener.Namespace == dep.Namespace {
			listener.NotifyChan <- e
		}
	}
}

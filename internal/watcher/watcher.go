package watcher

import (
	"errors"
	"github.com/golang/glog"
	"github.com/yarntime/hybridjob/pkg/types"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/pkg/api/v1"
	app "k8s.io/client-go/pkg/apis/apps/v1beta1"
	batchv1 "k8s.io/client-go/pkg/apis/batch/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"river/client"
	. "river/internal/model"
	"sync"
)

type WatchController struct {
	client *client.Client

	listeners map[string][]*Listener

	replicationControllerWatch watch.Interface

	deploymentWatch watch.Interface

	jobWatch watch.Interface

	daemonSetWatch watch.Interface

	statefulSetWatch watch.Interface

	persistentVolumeClaimWatch watch.Interface

	hybridJobWatch watch.Interface

	// This mutex guards all fields within this WatchController struct.
	mu sync.Mutex
}

func NewWatchController(client *client.Client) *WatchController {
	return &WatchController{
		client:    client,
		listeners: make(map[string][]*Listener),
	}
}

func (wc *WatchController) Register(resourceType string, listener *Listener) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	if _, ok := wc.listeners[resourceType]; !ok {
		wc.listeners[resourceType] = make([]*Listener, 0)
	}
	wc.listeners[resourceType] = append(wc.listeners[resourceType], listener)
	glog.Infof("register: resourceType(%s), target(%v)", resourceType, listener.Target)
	return nil
}

func (wc *WatchController) UnRegister(resourceType string, listener *Listener) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	if _, ok := wc.listeners[resourceType]; !ok {
		glog.Errorf("Unable to find the related listener.")
		return errors.New("Unable to find the related listener.")
	}

	for i := 0; i < len(wc.listeners[resourceType]); i++ {
		if wc.listeners[resourceType][i] == listener {
			wc.listeners[resourceType] = append(wc.listeners[resourceType][:i], wc.listeners[resourceType][i+1:]...)
			glog.Infof("unregister: resourceType(%s), target(%s)", resourceType, listener.Target)
			break
		}
	}
	return nil
}

func (wc *WatchController) Run() error {

	err := wc.InitWatcher()
	if err != nil {
		glog.Fatal("Failed to watch resource.")
		return err
	}

	for {
		select {
		case t, ok := <-wc.replicationControllerWatch.ResultChan():
			if !ok {
				glog.V(8).Infof("Renew Replication Controller watch.")
				wc.RenewWatch(ReplicationController)
				continue
			}
			rc := t.Object.(*v1.ReplicationController)
			target := GenerateTarget(rc.Name, rc.Namespace, rc.Labels)
			wc.Inform(target, ALL, t)
			wc.Inform(target, ReplicationController, t)
		case t, ok := <-wc.deploymentWatch.ResultChan():
			if !ok {
				glog.V(8).Infof("Renew Deployment watch.")
				wc.RenewWatch(Deployment)
				continue
			}
			dep := t.Object.(*v1beta1.Deployment)
			target := GenerateTarget(dep.Name, dep.Namespace, dep.Labels)
			wc.Inform(target, ALL, t)
			wc.Inform(target, Deployment, t)
		case t, ok := <-wc.jobWatch.ResultChan():
			if !ok {
				glog.V(8).Infof("Renew Job watch.")
				wc.RenewWatch(Job)
				continue
			}
			job := t.Object.(*batchv1.Job)
			target := GenerateTarget(job.Name, job.Namespace, job.Labels)
			wc.Inform(target, ALL, t)
			wc.Inform(target, Job, t)
		case t, ok := <-wc.daemonSetWatch.ResultChan():
			if !ok {
				glog.V(8).Infof("Renew DaemonSet watch.")
				wc.RenewWatch(DaemonSet)
				continue
			}
			ds := t.Object.(*v1beta1.DaemonSet)
			target := GenerateTarget(ds.Name, ds.Namespace, ds.Labels)
			wc.Inform(target, ALL, t)
			wc.Inform(target, DaemonSet, t)
		case t, ok := <-wc.statefulSetWatch.ResultChan():
			if !ok {
				glog.V(8).Infof("Renew StatefulSet watch.")
				wc.RenewWatch(StatefulSet)
				continue
			}
			ss := t.Object.(*app.StatefulSet)
			target := GenerateTarget(ss.Name, ss.Namespace, ss.Labels)
			wc.Inform(target, ALL, t)
			wc.Inform(target, StatefulSet, t)
		case t, ok := <-wc.persistentVolumeClaimWatch.ResultChan():
			if !ok {
				glog.V(8).Infof("Renew Persistent volume claim watch.")
				wc.RenewWatch(PersistentVolumeClaim)
				continue
			}
			pvc := t.Object.(*v1.PersistentVolumeClaim)
			target := GenerateTarget(pvc.Name, pvc.Namespace, pvc.Labels)
			wc.Inform(target, PersistentVolumeClaim, t)
		case t, ok := <-wc.hybridJobWatch.ResultChan():
			if !ok {
				glog.V(8).Infof("Renew hybrid job watch.")
				wc.RenewWatch(HybridJob)
				continue
			}
			hj := t.Object.(*types.HybridJob)
			target := GenerateTarget(hj.Name, hj.Namespace, hj.Labels)
			wc.Inform(target, HybridJob, t)
		}
	}
}

func GenerateTarget(name, namespace string, label map[string]string) Target {
	target := Target{}
	releaseName, ok := label["release"]
	if ok {
		target.ReleaseName = releaseName
	}
	target.ServiceName = namespace + "/" + name
	return target
}

func (wc *WatchController) Inform(target Target, resourceType string, event watch.Event) {
	if event.Type != watch.Modified {
		return
	}
	listeners, ok := wc.listeners[resourceType]
	if !ok {
		return
	}
	wc.mu.Lock()
	defer wc.mu.Unlock()

	for i := 0; i < len(listeners); i++ {
		if target.ReleaseName != "" && listeners[i].Target.ReleaseName == target.ReleaseName {
			glog.Infof("notify: resourceType(%s), release(%s)", resourceType, target.ReleaseName)
			listeners[i].NotifyChan <- event
		} else if target.ServiceName != "" && listeners[i].Target.ServiceName == target.ServiceName {
			glog.Infof("notify: resourceType(%s), service(%s)", resourceType, target.ServiceName)
			listeners[i].NotifyChan <- event
		}
	}
}

func (wc *WatchController) InitWatcher() error {
	wc.RenewWatch(ReplicationController)
	wc.RenewWatch(Deployment)
	wc.RenewWatch(Job)
	wc.RenewWatch(DaemonSet)
	wc.RenewWatch(StatefulSet)
	wc.RenewWatch(PersistentVolumeClaim)
	wc.RenewWatch(HybridJob)
	return nil
}

func (wc *WatchController) RenewWatch(resourceType string) error {

	listOptions := meta_v1.ListOptions{}

	switch resourceType {
	case ReplicationController:
		rw, err := wc.client.WatchAllReplicationControllers(listOptions)
		if err != nil {
			glog.Fatalf("Failed to watch replication controller. err: %v", err)
		}
		wc.replicationControllerWatch = rw
	case Deployment:
		dw, err := wc.client.WatchAllDeployments(listOptions)
		if err != nil {
			glog.Fatalf("Failed to watch deployments. err: %v", err)
		}
		wc.deploymentWatch = dw
	case Job:
		jw, err := wc.client.WatchAllJobs(listOptions)
		if err != nil {
			glog.Fatalf("Failed to watch jobs. err: %v", err)
			return err
		}
		wc.jobWatch = jw
	case DaemonSet:
		dsw, err := wc.client.WatchAllDaemonSets(listOptions)
		if err != nil {
			glog.Fatalf("Failed to watch daemonsets. err: %v", err)
		}
		wc.daemonSetWatch = dsw
	case StatefulSet:
		ssw, err := wc.client.WatchAllStatefulSets(listOptions)
		if err != nil {
			glog.Fatalf("Failed to watch statefulsets. err: %v", err)
		}
		wc.statefulSetWatch = ssw
	case PersistentVolumeClaim:
		pvcw, err := wc.client.WatchAllPersistentVolumeClaim(listOptions)
		if err != nil {
			glog.Fatalf("Failed to watch persistent volume claim. err: %v", err)
		}
		wc.persistentVolumeClaimWatch = pvcw
	case HybridJob:
		hjw, err := wc.client.WatchAllHybridJobs(listOptions)
		if err != nil {
			glog.Fatalf("Failed to watch hybrid job. err: %v", err)
		}
		wc.hybridJobWatch = hjw
	}
	return nil
}

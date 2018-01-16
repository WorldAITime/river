package app

import (
	"errors"
	"github.com/golang/glog"
	v1 "k8s.io/client-go/pkg/api/v1"
	app "k8s.io/client-go/pkg/apis/apps/v1beta1"
	batchv1 "k8s.io/client-go/pkg/apis/batch/v1"
	v1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/Masterminds/glide/action"
	"github.com/yarntime/hybridjob/pkg/tools"
	"github.com/yarntime/hybridjob/pkg/types"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"river/client"
	model "river/internal/model"
	"river/internal/watcher"
	"time"
)

var PING_PERIOD = 5 * time.Second

type AppController struct {
	watcher *watcher.WatchController
	client  *client.Client
}

func NewAppController(w *watcher.WatchController, c *client.Client) *AppController {
	return &AppController{
		watcher: w,
		client:  c,
	}
}

type Metadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"`

	Labels map[string]string `json:"labels,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty"`

	DesiredReplicas   *int32 `json:"desiredreplicas"`
	AvailableReplicas int32  `json:"availablereplicas"`
	ActiveReplicas    int32  `json:"activereplicas"`
	FailedReplicas    int32  `json:"failedreplicas"`
	MinReplicas       int32  `json:"minreplicas"`
	MaxReplicas       int32  `json:"maxreplicas"`
}

type App struct {
	Metadata  `json:"metadata"`
	Service   v1.Service `json:"service"`
	Completed bool       `json:"completed"`
}

type ReleaseResources struct {
	Apps []App `json:"apps"`
}

func (a *AppController) ServeRequest(ws *model.Websocket, namespace string, name string) error {

	labelSelector := labels.Set(map[string]string{"release": name}).AsSelector()
	listOptions := meta_v1.ListOptions{
		LabelSelector: labelSelector.String(),
	}

	resources := a.BuildResult(namespace, listOptions)
	ws.Conn.WriteJSON(resources)

	allWatch := &model.Listener{
		Target: model.Target{
			ReleaseName: name,
		},
		NotifyChan: make(chan watch.Event),
	}
	a.watcher.Register(model.ALL, allWatch)
	defer a.watcher.UnRegister(model.ALL, allWatch)

	ticker := time.NewTicker(PING_PERIOD)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-ws.CloseChan:
			glog.V(8).Info("Client connection is closed.")
			return nil
		case t := <-allWatch.NotifyChan:
			switch t.Object.(type) {
			case *v1.ReplicationController:
				rc := t.Object.(*v1.ReplicationController)
				for i := 0; i < len(resources.Apps); i++ {
					if resources.Apps[i].Metadata.Name == rc.Name && resources.Apps[i].Metadata.Type == model.ReplicationController {
						app := convertRcToApp(*rc)
						resources.Apps[i] = app
						err := ws.Conn.WriteJSON(resources)
						if err != nil {
							glog.Errorf("watch replication controller error, %v", err)
							return err
						}
						break
					}
				}
			case *v1beta1.Deployment:
				dep := t.Object.(*v1beta1.Deployment)
				for i := 0; i < len(resources.Apps); i++ {
					if resources.Apps[i].Metadata.Name == dep.Name && resources.Apps[i].Metadata.Type == model.Deployment {
						app := convertDepToApp(*dep)
						resources.Apps[i] = app
						err := ws.Conn.WriteJSON(resources)
						if err != nil {
							glog.Errorf("watch deployment error, %v", err)
							return err
						}
						break
					}
				}
			case *batchv1.Job:
				job := t.Object.(*batchv1.Job)
				for i := 0; i < len(resources.Apps); i++ {
					if resources.Apps[i].Metadata.Name == job.Name && resources.Apps[i].Metadata.Type == model.Job {
						app := convertJobToApp(*job)
						resources.Apps[i] = app
						err := ws.Conn.WriteJSON(resources)
						if err != nil {
							glog.Errorf("watch job error, %v", err)
							return err
						}
						break
					}
				}
			case *v1beta1.DaemonSet:
				daemonSet := t.Object.(*v1beta1.DaemonSet)
				for i := 0; i < len(resources.Apps); i++ {
					if resources.Apps[i].Metadata.Name == daemonSet.Name && resources.Apps[i].Metadata.Type == model.DaemonSet {
						app := convertDaemonSetToApp(*daemonSet)
						resources.Apps[i] = app
						err := ws.Conn.WriteJSON(resources)
						if err != nil {
							glog.Errorf("watch daemonset error, %v", err)
							return err
						}
						break
					}
				}
			case *app.StatefulSet:
				statefulSet := t.Object.(*app.StatefulSet)
				for i := 0; i < len(resources.Apps); i++ {
					if resources.Apps[i].Metadata.Name == statefulSet.Name && resources.Apps[i].Metadata.Type == model.StatefulSet {
						app := convertStatefulSetToApp(*statefulSet)
						resources.Apps[i] = app
						err := ws.Conn.WriteJSON(resources)
						if err != nil {
							glog.Errorf("watch statefulset error, %v", err)
							return err
						}
						break
					}
				}
			case *types.HybridJob:
				hybridJob := t.Object.(*types.HybridJob)
				for i := 0; i < len(resources.Apps); i++ {
					if resources.Apps[i].Metadata.Name == hybridJob.Name && resources.Apps[i].Metadata.Type == model.StatefulSet {
						app := convertHybridJobToApp(*hybridJob)
						resources.Apps[i] = app
						err := ws.Conn.WriteJSON(resources)
						if err != nil {
							glog.Errorf("watch hybrid job error, %v", err)
							return err
						}
						break
					}
				}
			}
		case <-ticker.C:
			err := ws.Conn.WriteJSON(resources)
			if err != nil {
				glog.Error("Failed to send msg to clinet.")
				return err
			}

			if ws.NoReplyTimes >= 5 {
				glog.Errorf("Failed to get pong message from client for 5 times, the connection may be disconnected.")
				return errors.New("Failed to get pong message from client for 5 times, the connection may be disconnected.")
			}
			ws.NoReplyTimes++
		}
	}
}

func (a *AppController) BuildResult(namespace string, listOptions meta_v1.ListOptions) ReleaseResources {

	resources := ReleaseResources{}

	nameToService := make(map[string]v1.Service)
	sl, _ := a.client.ListServices(namespace, listOptions)

	for i := 0; i < len(sl.Items); i++ {
		nameToService[sl.Items[i].Name] = sl.Items[i]
	}

	var apps []App

	rcAppChan := make(chan []App)
	depAppChan := make(chan []App)
	jobAppChan := make(chan []App)
	daemonSetAppChan := make(chan []App)
	statefulSetAppChan := make(chan []App)
	hybridJobAppChan := make(chan []App)

	go a.GenerateRCApps(namespace, listOptions, nameToService, rcAppChan)
	go a.GenerateDeploymentApps(namespace, listOptions, nameToService, depAppChan)
	go a.GenerateJobApps(namespace, listOptions, nameToService, jobAppChan)
	go a.GenerateDaemonSetApps(namespace, listOptions, nameToService, daemonSetAppChan)
	go a.GenerateStatefulSetApps(namespace, listOptions, nameToService, statefulSetAppChan)
	go a.GenerateHybridJobApps(namespace, listOptions, nameToService, hybridJobAppChan)

	apps = <-rcAppChan
	resources.Apps = append(resources.Apps, apps...)
	apps = <-depAppChan
	resources.Apps = append(resources.Apps, apps...)
	apps = <-jobAppChan
	resources.Apps = append(resources.Apps, apps...)
	apps = <-daemonSetAppChan
	resources.Apps = append(resources.Apps, apps...)
	apps = <-statefulSetAppChan
	resources.Apps = append(resources.Apps, apps...)
	apps = <-hybridJobAppChan
	resources.Apps = append(resources.Apps, apps...)

	return resources
}

func (a *AppController) GenerateRCApps(namespace string, listOptions meta_v1.ListOptions, nameToService map[string]v1.Service, appChan chan []App) {
	// add replication controllers to resources
	apps := []App{}
	replicationControllers, _ := a.client.ListReplicationControllers(namespace, listOptions)
	for i := 0; i < len(replicationControllers.Items); i++ {
		rc := replicationControllers.Items[i]
		app := convertRcToApp(rc)

		if s, ok := nameToService[rc.Name]; ok {
			app.Service = s
		}

		apps = append(apps, app)
	}
	appChan <- apps
}

func convertRcToApp(rc v1.ReplicationController) App {
	return App{
		Metadata: Metadata{
			Name:              rc.Name,
			Namespace:         rc.Namespace,
			Type:              model.ReplicationController,
			DesiredReplicas:   rc.Spec.Replicas,
			AvailableReplicas: rc.Status.AvailableReplicas,
			ActiveReplicas:    rc.Status.AvailableReplicas,
			Labels:            rc.Labels,
			Annotations:       rc.Annotations,
		},
		Completed: false,
	}
}

func (a *AppController) GenerateDeploymentApps(namespace string, listOptions meta_v1.ListOptions, nameToService map[string]v1.Service, appChan chan []App) {
	// add deployments to resources
	apps := []App{}
	deployments, _ := a.client.ListDeployments(namespace, listOptions)
	for i := 0; i < len(deployments.Items); i++ {
		dep := deployments.Items[i]
		app := convertDepToApp(dep)

		if s, ok := nameToService[dep.Name]; ok {
			app.Service = s
		}

		apps = append(apps, app)
	}
	appChan <- apps
}

func convertDepToApp(dep v1beta1.Deployment) App {
	return App{
		Metadata: Metadata{
			Name:              dep.Name,
			Namespace:         dep.Namespace,
			Type:              model.Deployment,
			DesiredReplicas:   dep.Spec.Replicas,
			AvailableReplicas: dep.Status.AvailableReplicas,
			ActiveReplicas:    dep.Status.AvailableReplicas,
			Labels:            dep.Labels,
			Annotations:       dep.Annotations,
		},
		Completed: false,
	}
}

func (a *AppController) GenerateJobApps(namespace string, listOptions meta_v1.ListOptions, nameToService map[string]v1.Service, appChan chan []App) {
	// add batch jobs to resources
	apps := []App{}
	jobs, _ := a.client.ListJobs(namespace, listOptions)
	for i := 0; i < len(jobs.Items); i++ {
		job := jobs.Items[i]
		app := convertJobToApp(job)

		if s, ok := nameToService[job.Name]; ok {
			app.Service = s
		}

		apps = append(apps, app)
	}
	appChan <- apps
}

func convertJobToApp(job batchv1.Job) App {
	app := App{
		Metadata: Metadata{
			Name:              job.Name,
			Namespace:         job.Namespace,
			Type:              model.Job,
			DesiredReplicas:   job.Spec.Completions,
			AvailableReplicas: job.Status.Succeeded,
			ActiveReplicas:    job.Status.Active,
			FailedReplicas:    job.Status.Failed,
			Labels:            job.Labels,
			Annotations:       job.Annotations,
		},
		Completed: false,
	}

	len := len(job.Status.Conditions)
	for i := 0; i < len; i++ {
		c := job.Status.Conditions[i]
		if c.Type == batchv1.JobComplete && c.Status == v1.ConditionTrue {
			app.Completed = true
		}
	}
	return app
}

func (a *AppController) GenerateDaemonSetApps(namespace string, listOptions meta_v1.ListOptions, nameToService map[string]v1.Service, appChan chan []App) {
	// add daemonsets to resources
	apps := []App{}
	daemonSets, _ := a.client.ListDaemonSets(namespace, listOptions)
	for i := 0; i < len(daemonSets.Items); i++ {
		daemonSet := daemonSets.Items[i]
		app := convertDaemonSetToApp(daemonSet)

		if s, ok := nameToService[daemonSet.Name]; ok {
			app.Service = s
		}

		apps = append(apps, app)
	}
	appChan <- apps
}

func convertDaemonSetToApp(daemonSet v1beta1.DaemonSet) App {
	return App{
		Metadata: Metadata{
			Name:              daemonSet.Name,
			Namespace:         daemonSet.Namespace,
			Type:              model.DaemonSet,
			DesiredReplicas:   &daemonSet.Status.DesiredNumberScheduled,
			AvailableReplicas: daemonSet.Status.NumberReady,
			ActiveReplicas:    daemonSet.Status.NumberReady,
			Labels:            daemonSet.Labels,
			Annotations:       daemonSet.Annotations,
		},
		Completed: false,
	}
}

func (a *AppController) GenerateStatefulSetApps(namespace string, listOptions meta_v1.ListOptions, nameToService map[string]v1.Service, appChan chan []App) {
	// add statefulset to resources
	apps := []App{}
	statefulSets, _ := a.client.ListStatefulSets(namespace, listOptions)
	for i := 0; i < len(statefulSets.Items); i++ {
		statefulSet := statefulSets.Items[i]
		app := convertStatefulSetToApp(statefulSet)

		if s, ok := nameToService[statefulSet.Name]; ok {
			app.Service = s
		}

		apps = append(apps, app)
	}
	appChan <- apps
}

func convertStatefulSetToApp(statefulSet app.StatefulSet) App {
	return App{
		Metadata: Metadata{
			Name:              statefulSet.Name,
			Namespace:         statefulSet.Namespace,
			Type:              model.StatefulSet,
			DesiredReplicas:   statefulSet.Spec.Replicas,
			AvailableReplicas: statefulSet.Status.Replicas,
			ActiveReplicas:    statefulSet.Status.Replicas,
			Labels:            statefulSet.Labels,
			Annotations:       statefulSet.Annotations,
		},
		Completed: false,
	}
}

func (a *AppController) GenerateHybridJobApps(namespace string, listOptions meta_v1.ListOptions, nameToService map[string]v1.Service, appChan chan []App) {
	// add hybrid job to resources
	apps := []App{}
	hybridJobs, _ := a.client.ListHybridJobs(namespace, listOptions)
	for i := 0; i < len(hybridJobs.Items); i++ {
		hybridJob := hybridJobs.Items[0]
		app := convertHybridJobToApp(hybridJob)

		if s, ok := nameToService[hybridJob.Name]; ok {
			app.Service = s
		}

		apps = append(apps, app)
	}
	appChan <- apps
}

func convertHybridJobToApp(hybridJob types.HybridJob) App {
	app := App{
		Metadata: Metadata{
			Name:            hybridJob.Name,
			Namespace:       hybridJob.Namespace,
			Type:            model.HybridJob,
			Labels:          hybridJob.Labels,
			Annotations:     hybridJob.Annotations,
			DesiredReplicas: tools.NewInt32(0),
		},
		Completed: hybridJob.Status.Phase == types.Finished,
	}
	for _, tfReplicas := range hybridJob.Spec.ReplicaSpecs {
		app.MinReplicas += *tfReplicas.MinReplicas
		app.MaxReplicas += *tfReplicas.MaxReplicas
		if hybridJob.Status.TfReplicaStatus != nil {
			status, ok := hybridJob.Status.TfReplicaStatus[tfReplicas.TfReplicaType]
			if ok && status.Desired != 0 {
				*app.DesiredReplicas += hybridJob.Status.TfReplicaStatus[tfReplicas.TfReplicaType].Desired
			} else {
				*app.DesiredReplicas += *tfReplicas.MaxReplicas
			}
		} else {
			*app.DesiredReplicas += *tfReplicas.MaxReplicas
		}
	}
	return app
}

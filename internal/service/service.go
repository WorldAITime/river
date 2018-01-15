package service

import (
	"errors"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/watch"
	"river/client"
	model "river/internal/model"
	"river/internal/watcher"
	"time"
)

var UNKNOW_WATCH_TYPE = "unknow watch type"
var PING_PERIOD = 5 * time.Second

type ServiceController struct {
	watcher *watcher.WatchController
	client  *client.Client
}

func NewServiceController(w *watcher.WatchController, c *client.Client) *ServiceController {
	return &ServiceController{
		watcher: w,
		client:  c,
	}
}

func (s *ServiceController) ServeRequest(ws *model.Websocket, subType string, namespace string, name string) error {

	switch subType {
	case model.ReplicationController:
		original, err := s.client.GetReplicationController(namespace, name)
		if err != nil {
			glog.Errorf("Failed to get original replication controller(%s/%s).", namespace, name)
			return err
		}
		return s.handleWatch(ws, original, model.ReplicationController, namespace, name)
	case model.Deployment:
		original, err := s.client.GetDeployment(namespace, name)
		if err != nil {
			glog.Errorf("Failed to get original deployment(%s/%s).", namespace, name)
			return err
		}
		return s.handleWatch(ws, original, model.Deployment, namespace, name)
	case model.DaemonSet:
		original, err := s.client.GetDaemonSet(namespace, name)
		if err != nil {
			glog.Errorf("Failed to get original daemonset(%s/%s).", namespace, name)
			return err
		}
		return s.handleWatch(ws, original, model.DaemonSet, namespace, name)
	case model.StatefulSet:
		original, err := s.client.GetStatefulSet(namespace, name)
		if err != nil {
			glog.Errorf("Failed to get original statefulset(%s/%s).", namespace, name)
			return err
		}
		return s.handleWatch(ws, original, model.StatefulSet, namespace, name)
	case model.PersistentVolumeClaim:
		original, err := s.client.GetPersistentVolumeClaim(namespace, name)
		if err != nil {
			glog.Errorf("Failed to get original persistent volume claim(%s/%s).", namespace, name)
			return err
		}
		return s.handleWatch(ws, original, model.PersistentVolumeClaim, namespace, name)
	}
	return nil
}

func (s *ServiceController) handleWatch(ws *model.Websocket, original interface{}, resourceType, namespace, name string) error {

	err := ws.Conn.WriteJSON(model.GetResponse(original))
	if err != nil {
		glog.Error("Failed to send msg to client.")
		return err
	}
	listener := &model.Listener{
		Target: model.Target{
			ServiceName: namespace + "/" + name,
		},
		NotifyChan: make(chan watch.Event),
	}

	s.watcher.Register(resourceType, listener)
	defer s.watcher.UnRegister(resourceType, listener)

	ticker := time.NewTicker(PING_PERIOD)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-ws.CloseChan:
			glog.V(8).Info("Client connection is closed.")
			return nil
		case t := <-listener.NotifyChan:
			if t.Type == "" {
				glog.Error(UNKNOW_WATCH_TYPE)
				return errors.New(UNKNOW_WATCH_TYPE)
			}
			if t.Type == watch.Modified {
				original = t.Object
				err := ws.Conn.WriteJSON(model.GetResponse(original))
				if err != nil {
					glog.Error("Failed to send msg to client.")
					return err
				}
			}
		case <-ticker.C:
			err := ws.Conn.WriteJSON(model.GetResponse(original))
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

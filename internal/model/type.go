package api

import (
	"github.com/golang/glog"
	ws "github.com/yarntime/websocket"
	. "k8s.io/apimachinery/pkg/watch"
)

const (
	ALL                   = "*"
	ReplicationController = "replicationcontrollers"
	Deployment            = "deployments"
	Job                   = "jobs"
	DaemonSet             = "daemonsets"
	StatefulSet           = "statefulsets"
	PersistentVolumeClaim = "persistentvolumeclaim"
	HybridJob             = "hybridjobs"
)

type Request struct {
	Type      string `json:"type"`
	SubType   string `json:"subtype"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	SubName   string `json:"subname"`
}

type Target struct {
	ReleaseName string
	ServiceName string
}

type Listener struct {
	Target     Target
	NotifyChan chan Event
}

type Response struct {
	Code    string      `json:"code"`
	Result  interface{} `json:"result"`
	Version string      `json:"version"`
}

func GetResponse(r interface{}) Response {
	return Response{
		Code:    "SUCCESS",
		Result:  r,
		Version: "1",
	}
}

type Websocket struct {
	Conn         *ws.Conn
	NoReplyTimes int
	CloseChan    chan int
}

func (w *Websocket) HeartBeat() {
	w.Conn.SetCloseHandler(func(code int, text string) error {
		glog.V(8).Info("Recv close message, return.")
		w.CloseChan <- 1
		return nil
	})

	go func() {
		for {
			_, _, err := w.Conn.ReadMessage()
			if err != nil {
				return
			}
			glog.V(8).Info("Recv ping message from client, reset noReplyTimes.")
			w.NoReplyTimes = 0
		}
	}()
}

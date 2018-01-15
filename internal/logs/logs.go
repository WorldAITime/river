package logs

import (
	"bufio"
	"github.com/golang/glog"
	"github.com/yarntime/websocket"
	"k8s.io/client-go/pkg/api/v1"
	"river/client"
	model "river/internal/model"
	"time"
)

type LogController struct {
	client *client.Client
}

func NewLogController(c *client.Client) *LogController {
	return &LogController{
		client: c,
	}
}

func (lc *LogController) ServeRequest(ws *model.Websocket, namespace string, name string, subname string) error {

	isAlive := newBool(true)
	closeChan := make(chan bool)
	go lc.logContainer(ws, namespace, name, subname, closeChan, isAlive)

	for {
		var req model.Request
		// Read in a new message as JSON and map it to a Message object
		err := ws.Conn.ReadJSON(&req)
		if err != nil {
			glog.Errorf("error: %v", err)
			return err
		}
		glog.Info("isAlive %v", *isAlive)
		if *isAlive {
			glog.Info("closing previous gorouting.")
			closeChan <- true
			time.Sleep(1 * time.Second)
		}
		go lc.logContainer(ws, req.Namespace, req.Name, req.SubName, closeChan, isAlive)
	}
}

func (lc *LogController) logContainer(ws *model.Websocket, namespace string, name string, subname string, closeChan chan bool, isAlive *bool) {

	*isAlive = true
	defer func() {
		*isAlive = false
	}()

	req := lc.client.WatchLogs(namespace, name, &v1.PodLogOptions{
		Follow:       true,
		Timestamps:   false,
		Container:    subname,
		SinceSeconds: newInt64(120),
		TailLines:    newInt64(100),
	})

	stream, err := req.Stream()
	if err != nil {
		ws.Conn.WriteMessage(websocket.TextMessage, []byte("container is starting, please retry later."))
		glog.Errorf("Error opening stream to %s/%s: %s\n", namespace, name, subname)
		return
	}
	defer stream.Close()

	go func() {
		select {
		case <-ws.CloseChan:
			glog.V(8).Info("connection is closed, exit.")
			stream.Close()
		case <-closeChan:
			glog.V(8).Info("log watch is finished, exit.")
			stream.Close()
		}
	}()

	reader := bufio.NewReader(stream)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return
		}
		err = ws.Conn.WriteMessage(websocket.TextMessage, line)
		if err != nil {
			return
		}
	}
}

func newInt64(val int64) *int64 {
	p := new(int64)
	*p = val
	return p
}

func newBool(val bool) *bool {
	p := new(bool)
	*p = val
	return p
}

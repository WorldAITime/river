package controller

import (
	"async/client"
	"async/model"
	"async/util"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"reflect"
)

type DepController struct {
	client  *client.K8sClient
	watcher *util.WatchController
}

func NewDepController(c *client.K8sClient, w *util.WatchController) *DepController {
	return &DepController{
		client:  c,
		watcher: w,
	}
}

func (dc *DepController) HandleRequest(conn *websocket.Conn) error {

	var listener model.Listener
	err := conn.ReadJSON(&listener)
	if err != nil {
		fmt.Printf("Failed to parse request message.%v\n", err)
		return errors.New("Failed to parse request message.")
	}

	oriDep, err := dc.client.GetDeployments(listener.ResourceName, listener.Namespace)
	if err != nil {
		fmt.Printf("Failed to get deployments(%s/%s).\n", listener.Namespace, listener.ResourceName)
		return errors.New("Failed to get deployments.")
	}

	var oriMiniDep model.MiniDeployment
	bytes, err := json.Marshal(oriDep)

	json.Unmarshal(bytes, &oriMiniDep)

	conn.WriteJSON(model.InformMessage{
		Resource: oriMiniDep,
	})

	notifyChan := make(chan watch.Event)
	listener.NotifyChan = notifyChan

	closeChan := make(chan int)

	defer close(closeChan)

	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				closeChan <- 1
				return
			}
			fmt.Printf("recv message: %s\n", string(msg))
		}
	}()

	dc.watcher.Register(&listener)
	defer dc.watcher.UnRegister(&listener)

	for {
		select {
		case t := <-notifyChan:
			dep := t.Object.(*v1beta1.Deployment)
			bytes, err = json.Marshal(dep)
			var miniDep model.MiniDeployment
			json.Unmarshal(bytes, &miniDep)
			if !reflect.DeepEqual(miniDep, oriMiniDep) {
				oriMiniDep = miniDep
				conn.WriteJSON(model.InformMessage{
					Resource:  oriMiniDep,
					EventType: string(t.Type),
				})
			}
		case <-closeChan:
			fmt.Println("Conn is closed by client.")
			return nil
		}
	}

	return nil
}

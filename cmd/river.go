package main

import (
	"net/http"

	"flag"
	"github.com/golang/glog"
	"github.com/yarntime/websocket"
	"river/client"
	"river/internal/app"
	"river/internal/logs"
	model "river/internal/model"
	"river/internal/service"
	"river/internal/watcher"
	wsController "river/internal/webconsole"
)

const (
	APP        = "app"
	SERVICE    = "service"
	POD        = "pod"
	WEB_SOCKET = "webconsole"
	LOGS       = "logs"
)

// Configure the upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var watchController *watcher.WatchController

type Controller struct {
	watchController   *watcher.WatchController
	appController     *app.AppController
	serviceController *service.ServiceController
	wsController      *wsController.WebConsoleController
	logController     *logs.LogController
}

var controller *Controller

func init() {
	flag.Set("alsologtostderr", "true")
	flag.Set("v", "8")
	flag.Parse()
}

func main() {

	glog.Info("starting")
	// Configure route
	http.HandleFunc("/websocket", handleConnections)

	c := client.NewClint("192.168.254.45:8080")

	watchController = watcher.NewWatchController(c)
	go watchController.Run()

	controller = &Controller{
		watchController:   watchController,
		appController:     app.NewAppController(watchController, c),
		serviceController: service.NewServiceController(watchController, c),
		wsController:      wsController.NewWebConsoleController(c),
		logController:     logs.NewLogController(c),
	}

	// Start the server on localhost port 8000 and log any errors
	glog.Info("http server started on :8000")
	err := http.ListenAndServe("0.0.0.0:8000", nil)
	if err != nil {
		glog.Fatal("ListenAndServe: ", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		glog.Fatal(err)
	}
	// Make sure we close the connection when the function returns
	defer ws.Close()

	var req model.Request
	// Read in a new message as JSON and map it to a Message object
	err = ws.ReadJSON(&req)
	if err != nil {
		glog.Errorf("error: %v", err)
		return
	}

	websocket := &model.Websocket{
		Conn:      ws,
		CloseChan: make(chan int),
	}

	switch req.Type {
	case APP:
		websocket.HeartBeat()
		err := controller.appController.ServeRequest(websocket, req.Namespace, req.Name)
		if err != nil {
			glog.Errorf("app error: %v", err)
			ws.WriteJSON(err)
			return
		}
		break
	case SERVICE:
		websocket.HeartBeat()
		err := controller.serviceController.ServeRequest(websocket, req.SubType, req.Namespace, req.Name)
		if err != nil {
			glog.Errorf("service error: %v", err)
			ws.WriteJSON(err)
			return
		}
		break
	case WEB_SOCKET:
		err := controller.wsController.ServeRequest(websocket, req.Namespace, req.Name, req.SubName)
		if err != nil {
			glog.Errorf("websocket error: %v", err)
			ws.WriteJSON(err)
			return
		}
		break
	case LOGS:
		err := controller.logController.ServeRequest(websocket, req.Namespace, req.Name, req.SubName)
		if err != nil {
			glog.Errorf("logs error: %v", err)
			ws.WriteJSON(err)
			return
		}
		break
	}
}

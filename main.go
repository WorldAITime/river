package main

import (
	"async/client"
	"async/controller"
	"async/util"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var depController *controller.DepController

func main() {

	k8sClient := client.NewK8sClint()

	wi, err := k8sClient.WatchDeployments()
	if err != nil {
		fmt.Printf("Failed to watch deployments.")
		os.Exit(1)
	}

	watchController := util.NewWatchController(wi)

	depController = controller.NewDepController(k8sClient, watchController)

	go watchController.Run()

	http.HandleFunc("/websocket", handleConnections)

	fmt.Println("http server started on :8000")
	err = http.ListenAndServe("0.0.0.0:8000", nil)
	if err != nil {
		fmt.Printf("ListenAndServe: %v\n", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		glog.Fatal(err)
	}

	defer ws.Close()

	depController.HandleRequest(ws)

}

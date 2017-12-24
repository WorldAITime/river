package webconsole

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/golang/glog"
	"github.com/yarntime/websocket"
	"io/ioutil"
	"net"
	"os"
	"river/client"
	model "river/internal/model"
)

var BAES_URL = "wss://%s/api/v1/namespaces/%s/pods/%s/exec?stdout=1&stdin=1&stderr=1&tty=1&command=/bin/sh&container=%s"

var KUBERNETES_SERVICE = ""

type WebConsoleController struct{}

func NewWebConsoleController(c *client.Client) *WebConsoleController {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) == 0 || len(port) == 0 {
		glog.Error("unable to load in-cluster configuration, KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined")
		return nil
	}
	KUBERNETES_SERVICE = net.JoinHostPort(host, port)
	return &WebConsoleController{}
}

func (wc *WebConsoleController) ServeRequest(ws *model.Websocket, namespace string, name string, subname string) error {

	server := fmt.Sprintf(BAES_URL, KUBERNETES_SERVICE, namespace, name, subname)

	rootCAFile := "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	CAData, err := ioutil.ReadFile(rootCAFile)
	if err != nil {
		glog.Fatal("Could not load server certificate!")
		return err
	}

	tlsConfig := &tls.Config{
		// Can't use SSLv3 because of POODLE and BEAST
		// Can't use TLSv1.0 because of POODLE and BEAST using CBC cipher
		// Can't use TLSv1.1 because of RC4 cipher usage
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
	}

	tlsConfig.RootCAs = rootCertPool(CAData)

	var dial = &websocket.Dialer{
		TLSClientConfig: tlsConfig,
		Subprotocols:    []string{"base64.channel.k8s.io", "channel.k8s.io"},
	}

	token, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return err
	}

	head := make(map[string][]string)
	head["Authorization"] = []string{"Bearer " + string(token)}

	wsToPod, _, err := dial.Dial(server, head)

	if err != nil {
		glog.Fatal(err)
	}

	defer wsToPod.Close()

	closePutChan := make(chan int)
	closeReadChan := make(chan int)

	go wc.ReadFromPod(ws.Conn, wsToPod, closeReadChan)

	go wc.PutToPod(ws.Conn, wsToPod, closePutChan)

	select {
	case <-closePutChan:
		closeReadChan <- 1
	}

	return nil
}

// rootCertPool returns nil if caData is empty.  When passed along, this will mean "use system CAs".
// When caData is not empty, it will be the ONLY information used in the CertPool.
func rootCertPool(caData []byte) *x509.CertPool {
	// What we really want is a copy of x509.systemRootsPool, but that isn't exposed.  It's difficult to build (see the go
	// code for a look at the platform specific insanity), so we'll use the fact that RootCAs == nil gives us the system values
	// It doesn't allow trusting either/or, but hopefully that won't be an issue
	if len(caData) == 0 {
		return nil
	}

	// if we have caData, use it
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caData)
	return certPool
}

func (wc *WebConsoleController) ReadFromPod(ws *websocket.Conn, wsToPod *websocket.Conn, ch chan int) error {
	for {
		select {
		case <-ch:
			return nil
		default:
			wsToPod.SetReadDeadline(time.Now().Add(10 * time.Second))
			_, msg, err := wsToPod.ReadMessage()
			if err != nil {
				if e, ok := err.(net.Error); ok && e.Timeout() {
					continue
				}
				glog.Errorf("Failed to read from pod: %s", err.Error())
				ws.Close()
				return nil
			}

			if len(msg) == 1 {
				continue
			}

			ws.WriteMessage(websocket.TextMessage, msg)
		}
	}
}

func (wc *WebConsoleController) PutToPod(ws *websocket.Conn, wsToPod *websocket.Conn, closeChan chan int) error {
	for {
		msgType, msg, err := ws.ReadMessage()
		if err != nil {
			glog.Errorf("Failed to read from client: %s", err.Error())
			closeChan <- 1
			return err
		}
		if msgType == websocket.CloseMessage {
			glog.V(8).Info("Recv close message, return.")
			closeChan <- 1
			return nil
		}

		wsToPod.WriteMessage(websocket.TextMessage, msg)
	}
}

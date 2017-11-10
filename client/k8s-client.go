package client

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sClient struct {
	clientset *k8s.Clientset
}

func NewK8sClint() *K8sClient {

	config, err := clientcmd.BuildConfigFromFlags("IP", "")
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := k8s.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	client := &K8sClient{
		clientset: clientset,
	}
	return client
}

func (c *K8sClient) WatchDeployments() (watch.Interface, error) {
	return c.clientset.ExtensionsV1beta1Client.Deployments("").Watch(v1.ListOptions{})
}

func (c *K8sClient) GetDeployments(name, namespace string) (*v1beta1.Deployment, error) {
	return c.clientset.ExtensionsV1beta1Client.Deployments(namespace).Get(name, v1.GetOptions{})
}

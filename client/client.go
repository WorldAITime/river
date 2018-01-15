package client

import (
	hybridJobClient "github.com/yarntime/hybridjob/pkg/client"
	"github.com/yarntime/hybridjob/pkg/types"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	watch "k8s.io/apimachinery/pkg/watch"
	k8s "k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/pkg/api/v1"
	appv1beta1 "k8s.io/client-go/pkg/apis/apps/v1beta1"
	batch "k8s.io/client-go/pkg/apis/batch/v1"
	v1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	k8sClient       *k8s.Clientset
	hybridJobClient *hybridJobClient.HybridJobClient
}

func NewClint(address string) *Client {

	return &Client{
		k8sClient:       NewK8sClint(address),
		hybridJobClient: hybridJobClient.NewHybridJobClient(address),
	}
}

func NewK8sClint(address string) *k8s.Clientset {

	config, err := getClientConfig(address)
	if err != nil {
		panic(err.Error())
	}

	// creates the clientSet
	clientSet, err := k8s.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientSet
}

func getClientConfig(host string) (*rest.Config, error) {
	if host != "" {
		return clientcmd.BuildConfigFromFlags(host, "")
	}
	return rest.InClusterConfig()
}

func (c *Client) WatchReplicationControllers(namespace string, listOption meta_v1.ListOptions) (watch.Interface, error) {
	return c.k8sClient.CoreV1().ReplicationControllers(namespace).Watch(listOption)
}

func (c *Client) WatchDeployments(namespace string, listOption meta_v1.ListOptions) (watch.Interface, error) {
	return c.k8sClient.ExtensionsV1beta1Client.Deployments(namespace).Watch(listOption)
}

func (c *Client) WatchJobs(namespace string, listOption meta_v1.ListOptions) (watch.Interface, error) {
	return c.k8sClient.BatchV1().Jobs(namespace).Watch(listOption)
}

func (c *Client) WatchAllReplicationControllers(listOption meta_v1.ListOptions) (watch.Interface, error) {
	return c.k8sClient.CoreV1().ReplicationControllers("").Watch(listOption)
}

func (c *Client) WatchAllDeployments(listOption meta_v1.ListOptions) (watch.Interface, error) {
	return c.k8sClient.ExtensionsV1beta1Client.Deployments("").Watch(listOption)
}

func (c *Client) WatchAllJobs(listOption meta_v1.ListOptions) (watch.Interface, error) {
	return c.k8sClient.BatchV1().Jobs("").Watch(listOption)
}

func (c *Client) WatchAllDaemonSets(listOption meta_v1.ListOptions) (watch.Interface, error) {
	return c.k8sClient.ExtensionsV1beta1Client.DaemonSets("").Watch(listOption)
}

func (c *Client) WatchAllStatefulSets(listOption meta_v1.ListOptions) (watch.Interface, error) {
	return c.k8sClient.AppsV1beta1Client.StatefulSets("").Watch(listOption)
}

func (c *Client) WatchAllPersistentVolumeClaim(listOption meta_v1.ListOptions) (watch.Interface, error) {
	return c.k8sClient.CoreV1Client.PersistentVolumeClaims("").Watch(listOption)
}

func (c *Client) WatchAllHybridJobs(listOption meta_v1.ListOptions) (watch.Interface, error) {
	return c.hybridJobClient.Watch(listOption)
}

func (c *Client) ListReplicationControllers(namespace string, listOption meta_v1.ListOptions) (*v1.ReplicationControllerList, error) {
	return c.k8sClient.CoreV1().ReplicationControllers(namespace).List(listOption)
}

func (c *Client) ListDeployments(namespace string, listOption meta_v1.ListOptions) (*v1beta1.DeploymentList, error) {
	return c.k8sClient.ExtensionsV1beta1Client.Deployments(namespace).List(listOption)
}

func (c *Client) ListJobs(namespace string, listOption meta_v1.ListOptions) (*batch.JobList, error) {
	return c.k8sClient.BatchV1().Jobs(namespace).List(listOption)
}

func (c *Client) ListServices(namespace string, listOption meta_v1.ListOptions) (*v1.ServiceList, error) {
	return c.k8sClient.CoreV1().Services(namespace).List(listOption)
}

func (c *Client) ListDaemonSets(namespace string, listOption meta_v1.ListOptions) (*v1beta1.DaemonSetList, error) {
	return c.k8sClient.ExtensionsV1beta1().DaemonSets(namespace).List(listOption)
}

func (c *Client) ListStatefulSets(namespace string, listOption meta_v1.ListOptions) (*appv1beta1.StatefulSetList, error) {
	return c.k8sClient.AppsV1beta1().StatefulSets(namespace).List(listOption)
}

func (c *Client) ListHybridJobs(namespace string, listOption meta_v1.ListOptions) (*types.HybridJobList, error) {
	return c.hybridJobClient.List(namespace, listOption)
}

func (c *Client) GetService(name string, namespace string) (*v1.Service, error) {
	return c.k8sClient.Services(namespace).Get(name, meta_v1.GetOptions{})
}

func (c *Client) ListPodsBySelector(namespace string, selector labels.Selector) (*v1.PodList, error) {
	lo := meta_v1.ListOptions{
		LabelSelector: selector.String(),
	}
	return c.k8sClient.Pods(namespace).List(lo)
}

func (c *Client) ListPodsByService(service *v1.Service) (*v1.PodList, error) {

	lo := meta_v1.ListOptions{
		LabelSelector: labels.Set(service.Spec.Selector).AsSelector().String(),
	}
	return c.k8sClient.Pods(service.Namespace).List(lo)
}

func (c *Client) GetReplicationController(namespace string, name string) (*v1.ReplicationController, error) {
	return c.k8sClient.ReplicationControllers(namespace).Get(name, meta_v1.GetOptions{})
}

func (c *Client) GetDeployment(namespace string, name string) (*v1beta1.Deployment, error) {
	return c.k8sClient.ExtensionsV1beta1Client.Deployments(namespace).Get(name, meta_v1.GetOptions{})
}

func (c *Client) GetDaemonSet(namespace string, name string) (*v1beta1.DaemonSet, error) {
	return c.k8sClient.ExtensionsV1beta1Client.DaemonSets(namespace).Get(name, meta_v1.GetOptions{})
}

func (c *Client) GetStatefulSet(namespace string, name string) (*appv1beta1.StatefulSet, error) {
	return c.k8sClient.AppsV1beta1Client.StatefulSets(namespace).Get(name, meta_v1.GetOptions{})
}

func (c *Client) GetPersistentVolumeClaim(namespace string, name string) (*v1.PersistentVolumeClaim, error) {
	return c.k8sClient.CoreV1Client.PersistentVolumeClaims(namespace).Get(name, meta_v1.GetOptions{})
}

func (c *Client) WatchLogs(namespace string, podName string, options *v1.PodLogOptions) *rest.Request {
	return c.k8sClient.CoreV1Client.Pods(namespace).GetLogs(podName, options)
}

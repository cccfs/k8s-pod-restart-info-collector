package main

import (
	"flag"
	"path/filepath"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

type NotifyOption struct {
	feishu Feishu
	slack Slack
	notify string
}

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		// use InClusterConfig
		config, err = rest.InClusterConfig()
		if err != nil {
			klog.Fatal(err)
		}
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

		
	var notifyChannel string
	flag.StringVar(&notifyChannel, "notify-channel", "feishu", "(optional) feishu,slack")
	no := &NotifyOption{}
	switch notifyChannel {
	case "feishu": 
		no.feishu = NewFeishu()
		no.notify = "feishu"
	case "slack":
		no.slack = NewSlack()
		no.notify = "slack"
	default:
		no.feishu = NewFeishu()
		no.notify = "feishu"
	}
	
	controller := NewController(clientset, *no)

	// Start the controller
	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(1, stop)

	// Wait forever
	select {}
}

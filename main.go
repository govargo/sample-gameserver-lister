package main

import (
	"flag"
	"log"
	"path/filepath"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var defaultKubeConfigPath string
	if home := homedir.HomeDir(); home != "" {
		// build kubeconfig path from $HOME dir
		defaultKubeConfigPath = filepath.Join(home, ".kube", "config")
	}

	// set kubeconfig flag
	kubeconfig := flag.String("kubeconfig", defaultKubeConfigPath, "kubeconfig config file")
	flag.Parse()
	config, _ := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	clientset, err := kubernetes.NewForConfig(config)
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	// Create InformerFactory
	informerFactory := informers.NewSharedInformerFactory(clientset, time.Second*30)
	agonesInformerFactory := externalversions.NewSharedInformerFactory(agonesClient, time.Second*30)

	// Create pod informer by informerFactory
	podInformer := informerFactory.Core().V1().Pods()

	// Create gameservers informer by informerFactory
	gameServers := agonesInformerFactory.Agones().V1().GameServers()
	gsInformer := gameServers.Informer()

	// Add EventHandler to informer
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(new interface{}) { log.Println("Pod Added") },
		UpdateFunc: func(old, new interface{}) { log.Println("Pod Updated") },
		DeleteFunc: func(old interface{}) { log.Println("Pod Deleted") },
	})
	gsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(new interface{}) { log.Println("GameServer Added") },
		UpdateFunc: func(old, new interface{}) { log.Println("GameServer Updated") },
		DeleteFunc: func(old interface{}) { log.Println("GameServer Deleted") },
	})

	// Start Go routines
	informerFactory.Start(wait.NeverStop)
	agonesInformerFactory.Start(wait.NeverStop)
	// Wait until finish caching with List API
	informerFactory.WaitForCacheSync(wait.NeverStop)
	agonesInformerFactory.WaitForCacheSync(wait.NeverStop)

	// Create Lister
	podLister := podInformer.Lister()
	gsLister := gameServers.Lister()

	for {
		// Get List of pods
		p := podLister.Pods("default")
		// Get List of gameservers
		gs, err := gsLister.List(labels.Everything())
		if err != nil {
			log.Fatal(err)
		}
		// Show gameserver's name & status & IPs
		for _, g := range gs {
			a, err := p.Get(g.GetName())
			if err != nil {
				log.Fatal(err)
			}
			log.Println("------------------------------")
			log.Println("Name: " + g.GetName())
			log.Println("Status: " + g.Status.State)
			log.Println("External IP: " + g.Status.Address)
			log.Println("Internal IP: " + a.Status.PodIP)
		}
		time.Sleep(time.Second * 25)
	}
}

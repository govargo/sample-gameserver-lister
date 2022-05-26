package main

import (
	"context"
	"flag"
	"path/filepath"
	"time"

	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
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
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	logger := runtime.NewLoggerWithSource("main")
	if err != nil {
		logger.WithError(err).Fatal("Could not create in cluster config")
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Fatal("Could not create the agones api clientset")
	}

	// Create InformerFactory which create the informer
	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Second*30)
	agonesInformerFactory := externalversions.NewSharedInformerFactory(agonesClient, time.Second*30)

	// Create Pod informer by informerFactory
	podInformer := informerFactory.Core().V1().Pods()

	// Create GameServer informer by informerFactory
	gameServers := agonesInformerFactory.Agones().V1().GameServers()
	gsInformer := gameServers.Informer()

	// Add EventHandler to informer
	// When the object's event happens, the function will be called
	// For example, when the pod is added, 'AddFunc' will be called and put out the "Pod Added"
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(new interface{}) { logger.Infof("Pod Added") },
		UpdateFunc: func(old, new interface{}) { logger.Infof("Pod Updated") },
		DeleteFunc: func(old interface{}) { logger.Infof("Pod Deleted") },
	})
	gsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(new interface{}) { logger.Infof("GameServer Added") },
		UpdateFunc: func(old, new interface{}) { logger.Infof("GameServer Updated") },
		DeleteFunc: func(old interface{}) { logger.Infof("GameServer Deleted") },
	})

	ctx := context.Background()

	// Start Go routines for informer
	informerFactory.Start(ctx.Done())
	agonesInformerFactory.Start(ctx.Done())
	// Wait until finish caching with List API
	informerFactory.WaitForCacheSync(ctx.Done())
	agonesInformerFactory.WaitForCacheSync(ctx.Done())

	// Create Lister which can list objects from the in-memory-cache
	podLister := podInformer.Lister()
	gsLister := gameServers.Lister()

	for {
		// Get List objects of Pods from Pod Lister
		p := podLister.Pods("default")
		// Get List objects of GameServers from GameServer Lister
		gs, err := gsLister.List(labels.Everything())
		if err != nil {
			panic(err)
		}
		// Show GameServer's name & status & IPs
		for _, g := range gs {
			a, err := p.Get(g.GetName())
			if err != nil {
				panic(err)
			}
			logger.Infof("------------------------------")
			logger.Infof("Name: %s", g.GetName())
			logger.Infof("Status: %s", g.Status.State)
			logger.Infof("External IP: %s", g.Status.Address)
			logger.Infof("Internal IP: %s", a.Status.PodIP)
		}
		time.Sleep(time.Second * 25)
	}
}

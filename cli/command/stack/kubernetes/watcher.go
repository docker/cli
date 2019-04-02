package kubernetes

import (
	"context"
	"sync"
	"time"

	apiv1beta1 "github.com/docker/cli/kubernetes/compose/v1beta1"
	"github.com/docker/cli/kubernetes/labels"
	"github.com/pkg/errors"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	podutils "k8s.io/kubernetes/pkg/api/v1/pod"
)

type stackListWatch interface {
	List(opts metav1.ListOptions) (*apiv1beta1.StackList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
}

type podListWatch interface {
	List(opts metav1.ListOptions) (*apiv1.PodList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
}

// DeployWatcher watches a stack deployement
type deployWatcher struct {
	pods   podListWatch
	stacks stackListWatch
}

// Watch watches a stuck deployement and return a chan that will holds the state of the stack
func (w *deployWatcher) Watch(name string, serviceNames []string, statusUpdates chan serviceStatus) error {
	errC := make(chan error, 1)
	defer close(errC)

	handlers := runtimeutil.ErrorHandlers

	// informer errors are reported using global error handlers
	runtimeutil.ErrorHandlers = append(handlers, func(err error) {
		errC <- err
	})
	defer func() {
		runtimeutil.ErrorHandlers = handlers
	}()

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	defer func() {
		cancel()
		wg.Wait()
	}()
	wg.Add(2)
	go func() {
		defer wg.Done()
		w.watchStackStatus(ctx, name, errC)
	}()
	go func() {
		defer wg.Done()
		w.waitForPods(ctx, name, serviceNames, errC, statusUpdates)
	}()

	return <-errC
}

type stackWatcher struct {
	resultChan chan error
	stackName  string
}

var _ cache.ResourceEventHandler = &stackWatcher{}

func (sw *stackWatcher) OnAdd(obj interface{}) {
	stack, ok := obj.(*apiv1beta1.Stack)
	switch {
	case !ok:
		sw.resultChan <- errors.Errorf("stack %s has incorrect type", sw.stackName)
	case stack.Status.Phase == apiv1beta1.StackFailure:
		sw.resultChan <- errors.Errorf("stack %s failed with status %s: %s", sw.stackName, stack.Status.Phase, stack.Status.Message)
	}
}

func (sw *stackWatcher) OnUpdate(oldObj, newObj interface{}) {
	sw.OnAdd(newObj)
}

func (sw *stackWatcher) OnDelete(obj interface{}) {
}

func (w *deployWatcher) watchStackStatus(ctx context.Context, stackname string, e chan error) {
	informer := newStackInformer(w.stacks, stackname)
	sw := &stackWatcher{
		resultChan: e,
	}
	informer.AddEventHandler(sw)
	informer.Run(ctx.Done())
}

type serviceStatus struct {
	name          string
	podsPending   int
	podsRunning   int
	podsSucceeded int
	podsFailed    int
	podsUnknown   int
	podsReady     int
	podsTotal     int
}

type podWatcher struct {
	stackName     string
	services      map[string]serviceStatus
	resultChan    chan error
	starts        map[string]int32
	indexer       cache.Indexer
	statusUpdates chan serviceStatus
}

var _ cache.ResourceEventHandler = &podWatcher{}

func (pw *podWatcher) handlePod(obj interface{}) {
	pod, ok := obj.(*apiv1.Pod)
	if !ok {
		pw.resultChan <- errors.Errorf("Pod has incorrect type in stack %s", pw.stackName)
		return
	}
	serviceName := pod.Labels[labels.ForServiceName]
	pw.updateServiceStatus(serviceName)
	if pw.allReady() {
		select {
		case pw.resultChan <- nil:
		default:
			// result has already been reported, just don't block
		}
	}
}

func (pw *podWatcher) updateServiceStatus(serviceName string) {
	pods, _ := pw.indexer.ByIndex("byservice", serviceName)
	status := serviceStatus{name: serviceName}
	for _, obj := range pods {
		if pod, ok := obj.(*apiv1.Pod); ok {
			switch pod.Status.Phase {
			case apiv1.PodPending:
				status.podsPending++
			case apiv1.PodRunning:
				status.podsRunning++
			case apiv1.PodSucceeded:
				status.podsSucceeded++
			case apiv1.PodFailed:
				status.podsFailed++
			case apiv1.PodUnknown:
				status.podsUnknown++
			}
			if podutils.IsPodReady(pod) {
				status.podsReady++
			}
		}
	}
	status.podsTotal = len(pods)
	oldStatus := pw.services[serviceName]
	if oldStatus != status {
		pw.statusUpdates <- status
	}
	pw.services[serviceName] = status
}

func (pw *podWatcher) allReady() bool {
	for _, status := range pw.services {
		if status.podsReady == 0 {
			return false
		}
	}
	return true
}

func (pw *podWatcher) OnAdd(obj interface{}) {
	pw.handlePod(obj)
}

func (pw *podWatcher) OnUpdate(oldObj, newObj interface{}) {
	pw.handlePod(newObj)
}

func (pw *podWatcher) OnDelete(obj interface{}) {
	pw.handlePod(obj)
}

func (w *deployWatcher) waitForPods(ctx context.Context, stackName string, serviceNames []string, e chan error, statusUpdates chan serviceStatus) {
	informer := newPodInformer(w.pods, stackName, cache.Indexers{
		"byservice": func(obj interface{}) ([]string, error) {
			pod, ok := obj.(*apiv1.Pod)
			if !ok {
				return nil, errors.Errorf("Pod has incorrect type in stack %s", stackName)
			}
			return []string{pod.Labels[labels.ForServiceName]}, nil
		}})
	services := map[string]serviceStatus{}
	for _, name := range serviceNames {
		services[name] = serviceStatus{name: name}
	}
	pw := &podWatcher{
		stackName:     stackName,
		services:      services,
		resultChan:    e,
		starts:        map[string]int32{},
		indexer:       informer.GetIndexer(),
		statusUpdates: statusUpdates,
	}
	informer.AddEventHandler(pw)
	informer.Run(ctx.Done())
}

func newPodInformer(podsClient podListWatch, stackName string, indexers cache.Indexers) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.LabelSelector = labels.SelectorForStack(stackName)
				options.IncludeUninitialized = true
				return podsClient.List(options)
			},

			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = labels.SelectorForStack(stackName)
				options.IncludeUninitialized = true
				return podsClient.Watch(options)
			},
		},
		&apiv1.Pod{},
		time.Second*5,
		indexers,
	)
}

func newStackInformer(stacksClient stackListWatch, stackName string) cache.SharedInformer {
	return cache.NewSharedInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = fields.OneTermEqualSelector("metadata.name", stackName).String()
				return stacksClient.List(options)
			},

			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.FieldSelector = fields.OneTermEqualSelector("metadata.name", stackName).String()
				return stacksClient.Watch(options)
			},
		},
		&apiv1beta1.Stack{},
		time.Second*5,
	)
}

package kubernetes

import (
	"testing"

	apiv1beta1 "github.com/docker/cli/kubernetes/compose/v1beta1"
	composelabels "github.com/docker/cli/kubernetes/labels"
	"github.com/gotestyourself/gotestyourself/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
)

var podsResource = apiv1.SchemeGroupVersion.WithResource("pods")
var podKind = apiv1.SchemeGroupVersion.WithKind("Pod")
var stacksResource = apiv1beta1.SchemeGroupVersion.WithResource("stacks")
var stackKind = apiv1beta1.SchemeGroupVersion.WithKind("Stack")

type testPodAndStackRepository struct {
	fake *k8stesting.Fake
}

func (r *testPodAndStackRepository) stackListWatchForNamespace(ns string) *testStackListWatch {
	return &testStackListWatch{fake: r.fake, ns: ns}
}
func (r *testPodAndStackRepository) podListWatchForNamespace(ns string) *testPodListWatch {
	return &testPodListWatch{fake: r.fake, ns: ns}
}

func newTestPodAndStackRepository(initialPods []apiv1.Pod, initialStacks []apiv1beta1.Stack, podWatchHandler, stackWatchHandler k8stesting.WatchReactionFunc) *testPodAndStackRepository {
	var scheme = runtime.NewScheme()
	var codecs = serializer.NewCodecFactory(scheme)
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
	apiv1.AddToScheme(scheme)
	apiv1beta1.AddToScheme(scheme)

	o := k8stesting.NewObjectTracker(scheme, codecs.UniversalDecoder())
	for _, obj := range initialPods {
		if err := o.Add(&obj); err != nil {
			panic(err)
		}
	}
	for _, obj := range initialStacks {
		if err := o.Add(&obj); err != nil {
			panic(err)
		}
	}
	fakePtr := &k8stesting.Fake{}
	fakePtr.AddReactor("*", "*", k8stesting.ObjectReaction(o))
	if podWatchHandler != nil {
		fakePtr.AddWatchReactor(podsResource.Resource, podWatchHandler)
	}
	if stackWatchHandler != nil {
		fakePtr.AddWatchReactor(stacksResource.Resource, stackWatchHandler)
	}
	fakePtr.AddWatchReactor("*", k8stesting.DefaultWatchReactor(watch.NewFake(), nil))
	return &testPodAndStackRepository{fake: fakePtr}
}

type testStackListWatch struct {
	fake *k8stesting.Fake
	ns   string
}

func (s *testStackListWatch) List(opts metav1.ListOptions) (*apiv1beta1.StackList, error) {
	obj, err := s.fake.Invokes(k8stesting.NewListAction(stacksResource, stackKind, s.ns, opts), &apiv1beta1.StackList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := k8stesting.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &apiv1beta1.StackList{}
	for _, item := range obj.(*apiv1beta1.StackList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}
func (s *testStackListWatch) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return s.fake.InvokesWatch(k8stesting.NewWatchAction(stacksResource, s.ns, opts))
}

type testPodListWatch struct {
	fake *k8stesting.Fake
	ns   string
}

func (p *testPodListWatch) List(opts metav1.ListOptions) (*apiv1.PodList, error) {
	obj, err := p.fake.Invokes(k8stesting.NewListAction(podsResource, podKind, p.ns, opts), &apiv1.PodList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := k8stesting.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &apiv1.PodList{}
	for _, item := range obj.(*apiv1.PodList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err

}
func (p *testPodListWatch) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return p.fake.InvokesWatch(k8stesting.NewWatchAction(podsResource, p.ns, opts))
}

func TestDeployWatchOk(t *testing.T) {
	stack := apiv1beta1.Stack{
		ObjectMeta: metav1.ObjectMeta{Name: "test-stack", Namespace: "test-ns"},
	}

	serviceNames := []string{"svc1", "svc2"}
	testRepo := newTestPodAndStackRepository(nil, []apiv1beta1.Stack{stack}, func(action k8stesting.Action) (handled bool, ret watch.Interface, err error) {
		res := watch.NewFake()
		go func() {
			pod1 := &apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test1",
					Namespace: "test-ns",
					Labels:    composelabels.ForService("test-stack", "svc1"),
				},
				Status: apiv1.PodStatus{
					Phase: apiv1.PodRunning,
					Conditions: []apiv1.PodCondition{
						{
							Type:   apiv1.PodReady,
							Status: apiv1.ConditionTrue,
						},
					},
				},
			}
			pod2 := &apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test2",
					Namespace: "test-ns",
					Labels:    composelabels.ForService("test-stack", "svc2"),
				},
				Status: apiv1.PodStatus{
					Phase: apiv1.PodRunning,
					Conditions: []apiv1.PodCondition{
						{
							Type:   apiv1.PodReady,
							Status: apiv1.ConditionTrue,
						},
					},
				},
			}
			res.Add(pod1)
			res.Add(pod2)
		}()

		return true, res, nil
	}, nil)

	testee := &deployWatcher{
		stacks: testRepo.stackListWatchForNamespace("test-ns"),
		pods:   testRepo.podListWatchForNamespace("test-ns"),
	}

	statusUpdates := make(chan serviceStatus)
	go func() {
		for range statusUpdates {
		}
	}()
	defer close(statusUpdates)
	err := testee.Watch(stack.Name, serviceNames, statusUpdates)
	assert.NilError(t, err)
}

func TestDeployReconcileFailure(t *testing.T) {
	stack := apiv1beta1.Stack{
		ObjectMeta: metav1.ObjectMeta{Name: "test-stack", Namespace: "test-ns"},
	}

	serviceNames := []string{"svc1", "svc2"}
	testRepo := newTestPodAndStackRepository(nil, []apiv1beta1.Stack{stack}, nil, func(action k8stesting.Action) (handled bool, ret watch.Interface, err error) {
		res := watch.NewFake()
		go func() {
			sfailed := stack
			sfailed.Status = apiv1beta1.StackStatus{
				Phase:   apiv1beta1.StackFailure,
				Message: "test error",
			}
			res.Modify(&sfailed)
		}()

		return true, res, nil
	})

	testee := &deployWatcher{
		stacks: testRepo.stackListWatchForNamespace("test-ns"),
		pods:   testRepo.podListWatchForNamespace("test-ns"),
	}

	statusUpdates := make(chan serviceStatus)
	go func() {
		for range statusUpdates {
		}
	}()
	defer close(statusUpdates)
	err := testee.Watch(stack.Name, serviceNames, statusUpdates)
	assert.ErrorContains(t, err, "Failure: test error")
}

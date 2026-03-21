package tools

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func newTestTools(objects ...corev1.Pod) *K8sTools {
	clientset := fake.NewSimpleClientset()
	for i := range objects {
		clientset.CoreV1().Pods(objects[i].Namespace).Create(
			context.Background(), &objects[i], metav1.CreateOptions{})
	}
	return &K8sTools{client: clientset}
}

func TestGetPodStatus_Empty(t *testing.T) {
	k := newTestTools()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"namespace": "default"}

	result, err := k.GetPodStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
}

func TestGetPodStatus_WithPods(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "nginx"}}},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	k := newTestTools(pod)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"namespace": "default"}

	result, err := k.GetPodStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
}

func TestGetNodeStatus(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	k := &K8sTools{client: clientset}
	req := mcp.CallToolRequest{}

	result, err := k.GetNodeStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
}

func TestGetEvents_NoWarnings(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	k := &K8sTools{client: clientset}
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"namespace": "", "limit": float64(10)}

	result, err := k.GetEvents(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
}

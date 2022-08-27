package pod

import (
	"context"
	"encoding/json"
	"fmt"
	CoreV1 "k8s.io/api/core/v1"
	MetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	K8sLabel "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func ListPod(client dynamic.Interface, labels map[string]string) []CoreV1.Pod {
	podList := []CoreV1.Pod{}
	utd, err := client.Resource(schema.GroupVersionResource{
		Version:  "v1",
		Resource: "pods",
	}).Namespace("default").List(context.TODO(), MetaV1.ListOptions{
		LabelSelector: K8sLabel.SelectorFromSet(labels).String(),
	})
	if err != nil {
		fmt.Println(err)
		return podList
	}

	podListObj := CoreV1.PodList{}

	podByte, err := utd.MarshalJSON()
	json.Unmarshal(podByte, &podListObj)
	for _, pod := range podListObj.Items {
		if pod.Status.Phase != CoreV1.PodRunning {
			podList = append(podList, pod)
		}
	}
	return podList
}

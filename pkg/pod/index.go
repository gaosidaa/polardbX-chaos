package pod

import (
	"ChaosApi/pkg/client"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

func GetLogs(podName string) (string, error) {

	var line int64 = 15
	req := client.ClientSet.CoreV1().Pods("default").GetLogs(podName, &CoreV1.PodLogOptions{
		TailLines: &line,
		Follow:    true,
	})
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		fmt.Println(err)
		return "", fmt.Errorf("error in opening stream")
	}

	var msg string
	for {
		buf := make([]byte, 2000)
		numBytes, err := podLogs.Read(buf)
		if numBytes == 0 {
			continue
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break

		}
		msg = string(buf[:numBytes])
		return msg, nil
	}

	return msg, nil
}

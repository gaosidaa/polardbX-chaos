package main

import (
	"ChaosApi/pkg/chaos"
	"ChaosApi/pkg/client"
	"ChaosApi/pkg/pdbx"
	"fmt"
	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"

	"log"
)

func main() {

	podLabels := make(map[string]string)
	podLabels["pdbx/cn-type"] = "rw"
	ns := []string{"default"}

	dynamicClient, err := client.InitDynamicClient("./configs/kube/config")
	if err != nil {
		log.Fatalln(err)
	}

	// 初始化一个空的workflow 第一个参数为父工作流的名称,第二个参数为父工作流的命名空间,第三个参数子工作流的名称
	workflow1 := chaos.NewWorkflowBase("serial-test", "default", "serial-pod").InjectDynamicClient(dynamicClient)

	// 初始化podChaos原子故障模版 默认类型为pod kill
	podKill := chaos.NewChaosTemplateBase("pod-kill", "100s", ns).
		SetLabels(podLabels).PodChaosInit()

	// 初始化podChaos原子故障模版 调用SetAction方法奖类型改成pod-failure
	podFailure := chaos.NewChaosTemplateBase("pod-failure", "100s", ns).
		SetLabels(podLabels).SetAction(string(v1alpha1.PodFailureAction)).
		SetMode(v1alpha1.OneMode).
		PodChaosInit()

	// 将已经初始化的两个故障模版注入到workflow 并设置type  以及父工作流deadline时间和子工作流deadline时间
	s1 := workflow1.InjectTemplate(
		podKill,
		podFailure,
	).SetWorkflowType(v1alpha1.TypeSerial, "250s", "200s")

	// 定义io workflow
	workflow2 := chaos.NewWorkflowBase("io-test-workflow", "default", "io-test").InjectDynamicClient(dynamicClient)
	// 初始化io 原子故障
	ioChaos := chaos.NewChaosTemplateBase("pod-failure", "100s", ns).
		SetLabels(podLabels).SetAction(string(v1alpha1.IoLatency)).
		SetMode(v1alpha1.OneMode).
		IoChaosInit("100ms", "/usr/share/sysbench/tests", 100)

	// 将io原子故障注入workflow
	s2 := workflow2.InjectTemplate(ioChaos).SetWorkflowType(v1alpha1.TypeSerial, "180s", "120s")

	// 初始化workflow
	workflow3 := chaos.NewWorkflowBase("network-workflow", "default", "network-error").InjectDynamicClient(dynamicClient)
	// 初始化网络原子故障
	networkChaos := chaos.NewChaosTemplateBase("etwork-task-in-serial-flow", "100s", ns).SetLabels(podLabels).SetMode(v1alpha1.OneMode).
		NetworkChaosInit(&v1alpha1.DelaySpec{
			Latency:     "20s",
			Correlation: "0",
			Jitter:      "0ms",
		})
	s3 := workflow3.InjectTemplate(networkChaos).SetWorkflowType(v1alpha1.TypeSerial, "200s", "120s")

	fmt.Println(s2.DeleteWorkflow(), s1.DeleteWorkflow())
	fmt.Println(s3.DeleteWorkflow())

	obj := pdbx.NewPolarDB("polardb-x").InjectDynamicClient(dynamicClient)
	obj.InitOjb()
	obj.CreatePolarDB()

}

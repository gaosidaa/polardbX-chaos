package main

import (
	"ChaosApi/pkg/chaos"
	"ChaosApi/pkg/client"
	"ChaosApi/pkg/pdbx"
	"ChaosApi/pkg/pod"
	"ChaosApi/pkg/sysbench"
	"fmt"
	"github.com/alibaba/polardbx-operator/api/v1/polardbx"
	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"log"
	"os"
	"time"
)

func main() {

	podLabels := make(map[string]string)
	podLabels["polardbx/cn-type"] = "rw"
	ns := []string{"default"}

	dynamicClient, err := client.InitDynamicClient("./configs/kube/config")
	if err != nil {
		log.Fatalln(err)
	}

	obj := pdbx.NewPolarDbBase("polardb-x").InjectDynamicClient(dynamicClient)

	if obj.CreatePolarDB() != nil {
		fmt.Printf("创建集群失败, err : %s", err)
		return
	}

	var star int
	for {
		star++
		if obj.GetStatus() == polardbx.PhaseRunning {
			fmt.Println("集群创建成功")
			break
		}
		time.Sleep(1 * time.Minute)

		if star > 15 {
			os.Exit(1)
			fmt.Printf("集群异常, err : %s", err)
			break
		}

	}

	bench2 := sysbench.NewSysbench("sysbench-prepare").InjectDynamicClient(dynamicClient)
	bench2.SetHost("polardb-x").SetArgs([]string{
		"--db-driver=mysql",
		"--mysql-host=$(POLARDB_X_SERVICE_HOST)",
		"--mysql-port=$(POLARDB_X_SERVICE_PORT)",
		"--mysql-user=$(POLARDB_X_USER)",
		"--mysql_password=$(POLARDB_X_PASSWD)",
		"--mysql-db=sysbench_test",
		"--mysql-table-engine=innodb",
		"--rand-init=on",
		"--max-requests=1",
		"--oltp-tables-count=1",
		"--report-interval=5",
		"--oltp-table-size=160000",
		"--oltp_skip_trx=on",
		"--oltp_auto_inc=off",
		"--oltp_secondary",
		"--oltp_range_size=5",
		"--mysql_table_options=dbpartition by hash(`id`)",
		"--num-threads=1",
		"--time=3600",
		"/usr/share/sysbench/tests/include/oltp_legacy/parallel_prepare.lua",
		"run",
	}).InitDatabase()

	if bench2.Create() != nil {
		fmt.Printf("sysbench-prepare 注入失败 ERR :%s", err)
	}

	fmt.Println("sysbench-prepare 注入成功")
	for {
		star++
		if bench2.IsComplete() {
			fmt.Println("数据注入成功")
			break
		}
		time.Sleep(1 * time.Minute)

		if star > 15 {
			fmt.Printf("sysbench 数据注入失败")
			os.Exit(1)
		}
	}

	fmt.Println("开始注入 sysbench-oltp-test ")
	bench := sysbench.NewSysbench("sysbench-oltp-test").InjectDynamicClient(dynamicClient)
	bench.SetHost("polardb-x").SetArgs([]string{
		"--db-driver=mysql",
		"--mysql-host=$(POLARDB_X_SERVICE_HOST)",
		"--mysql-port=$(POLARDB_X_SERVICE_PORT)",
		"--mysql-user=$(POLARDB_X_USER)",
		"--mysql_password=$(POLARDB_X_PASSWD)",
		"--mysql-db=sysbench_test",
		"--mysql-table-engine=innodb",
		"--rand-init=on",
		"--max-requests=0",
		"--oltp-tables-count=1",
		"--report-interval=5",
		"--oltp-table-size=160000",
		"--oltp_skip_trx=on",
		"--oltp_auto_inc=off",
		"--oltp_secondary",
		"--oltp_range_size=5",
		"--mysql-ignore-errors=all",
		"--num-threads=16",
		"--time=3600000",
		"/usr/share/sysbench/tests/include/oltp_legacy/oltp.lua",
		"run",
	})

	if bench.Create() != nil {
		fmt.Printf("sysbench-oltp-test 注入失败 ERR :%s", err)
	}

	for {
		time.Sleep(1 * time.Minute)
		star++
		if star > 15 {
			break
		}
		if bench.GetPhase() == corev1.PodRunning {
			podName, err := bench.GetPodName()
			if err != nil {
				fmt.Println(err)
				break
			}

			time.Sleep(15 * time.Second)

			log, err := pod.GetLogs(podName)
			if err != nil {
				fmt.Println(err)
				break
			}
			fmt.Printf(" start qps %s \n : \n", log)
			break
		}
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

	if s1.CreateWorkflow() != nil {
		fmt.Printf("WorkflowType 注入失败 ERR :%s", err)
	}

	for {
		time.Sleep(1 * time.Minute)

		if s1.GetStatus() {
			break
		}
		if star > 15 {
			break
		}
	}

	podName, err := bench.GetPodName()
	log, err := pod.GetLogs(podName)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("end sqs \n %s \n", log)

	errPod := pod.ListPod(dynamicClient, map[string]string{
		"polardbx/name": "polardb-x",
	})

	if len(errPod) == 0 {
		fmt.Println("全部pod恢复正常")
	} else {
		for _, pod := range errPod {
			fmt.Printf("%s 未恢复正常", pod)
		}
	}

	//fmt.Println(obj.DeletePolarDB(),
	//	workflow1.DeleteWorkflow(),
	//	bench.Delete(),
	//	bench2.Delete())
	obj.DeletePolarDB()
	workflow1.DeleteWorkflow()
	bench.Delete()
	bench2.Delete()


}

// 定义io workflow
//workflow2 := chaos.NewWorkflowBase("io-test-workflow", "default", "io-test").InjectDynamicClient(dynamicClient)
// 初始化io 原子故障
//ioChaos := chaos.NewChaosTemplateBase("pod-failure", "100s", ns).
//	SetLabels(podLabels).SetAction(string(v1alpha1.IoLatency)).
//	SetMode(v1alpha1.OneMode).
//	IoChaosInit("100ms", "/usr/share/sysbench/tests", 100)

// 将io原子故障注入workflow
//s2 := workflow2.InjectTemplate(ioChaos).SetWorkflowType(v1alpha1.TypeSerial, "180s", "120s")

// 初始化workflow
//workflow3 := chaos.NewWorkflowBase("network-workflow", "default", "network-error").InjectDynamicClient(dynamicClient)
// 初始化网络原子故障
//networkChaos := chaos.NewChaosTemplateBase("etwork-task-in-serial-flow", "100s", ns).SetLabels(podLabels).SetMode(v1alpha1.OneMode).
//	NetworkChaosInit(&v1alpha1.DelaySpec{
//		Latency:     "20s",
//		Correlation: "0",
//		Jitter:      "0ms",
//	})
//s3 := workflow3.InjectTemplate(networkChaos).SetWorkflowType(v1alpha1.TypeSerial, "200s", "120s")

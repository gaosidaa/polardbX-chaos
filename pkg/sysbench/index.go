package sysbench

import (
	"ChaosApi/pkg/client"
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	BatchV1 "k8s.io/api/batch/v1"
	CoreV1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	MetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"log"
)

type Sysbench struct {
	Name         string
	BackoffLimit int32
	Host         string
	Port         string
	Args         []string
	Client       dynamic.Interface
}

func NewSysbench(name string) *Sysbench {
	return &Sysbench{Name: name}
}

func (this *Sysbench) InjectDynamicClient(client dynamic.Interface) *Sysbench {
	this.Client = client
	return this
}

func NewJobs(obj *Sysbench) *BatchV1.Job {
	return &BatchV1.Job{
		TypeMeta: MetaV1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},
		ObjectMeta: MetaV1.ObjectMeta{
			Name:      obj.Name,
			Namespace: "default",
		},
		Spec: BatchV1.JobSpec{
			BackoffLimit: &obj.BackoffLimit,
			Template: CoreV1.PodTemplateSpec{
				Spec: CoreV1.PodSpec{
					RestartPolicy: CoreV1.RestartPolicyNever,
					Containers: []CoreV1.Container{
						{
							Name:  obj.Name,
							Image: "severalnines/sysbench",
							Env: []CoreV1.EnvVar{
								{
									Name:  "POLARDB_X_SERVICE_HOST",
									Value: obj.Host,
								},
								{
									Name:  "POLARDB_X_SERVICE_PORT",
									Value: "3306",
								},
								{
									Name:  "POLARDB_X_USER",
									Value: "root",
								},
								{
									Name:  "POLARDB_X_PASSWD",
									Value: "pgzxtppml",
								},
							},
							Command: []string{"sysbench"},
							Args:    obj.Args,
						},
					},
				},
			},
		},
	}
}
func (this *Sysbench) SetArgs(args []string) *Sysbench {
	this.Args = args
	return this
}

func (this *Sysbench) Create() error {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(NewJobs(this))
	if err != nil {
		log.Fatalln(err)
	}
	utd := unstructured.Unstructured{
		Object: obj,
	}

	_, err = this.Client.Resource(schema.GroupVersionResource{
		Group:    "batch",
		Version:  "v1",
		Resource: "jobs",
	}).Namespace(utd.GetNamespace()).Get(context.TODO(), utd.GetName(), MetaV1.GetOptions{})

	if apierrors.IsNotFound(err) {
		_, err = client.DynamicClient.Resource(schema.GroupVersionResource{
			Group:    "batch",
			Version:  "v1",
			Resource: "jobs",
		}).Namespace(utd.GetNamespace()).Create(context.TODO(), &utd, MetaV1.CreateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *Sysbench) GetPodName() (string, error) {
	jobs, err := client.ClientSet.BatchV1().Jobs("default").Get(context.TODO(), this.Name, MetaV1.GetOptions{})
	if err != nil {
		return "", err
	}
	pods, err := client.ClientSet.CoreV1().Pods("default").List(context.TODO(), MetaV1.ListOptions{
		LabelSelector: labels.FormatLabels(jobs.Labels),
	})
	if err != nil {
		return "", err
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("pod还未创建")
	}
	return pods.Items[0].Name, nil

}

func (this *Sysbench) GetPhase() CoreV1.PodPhase {
	jobs, err := client.ClientSet.BatchV1().Jobs("default").Get(context.TODO(), this.Name, MetaV1.GetOptions{})
	if err != nil {

		return ""
	}
	pods, err := client.ClientSet.CoreV1().Pods("default").List(context.TODO(), MetaV1.ListOptions{
		LabelSelector: labels.FormatLabels(jobs.Labels),
	})
	if err != nil {

		return ""
	}
	return pods.Items[0].Status.Phase

}

func (this *Sysbench) Delete() error {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(NewJobs(this))
	if err != nil {
		log.Fatalln(err)
	}
	utd := unstructured.Unstructured{
		Object: obj,
	}
	podName, err := this.GetPodName()
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("获取pod ERR:%v", err)
	}
	err = client.ClientSet.CoreV1().Pods(utd.GetNamespace()).Delete(context.TODO(), podName, MetaV1.DeleteOptions{})
	if err != nil {
		return err
	}

	err = this.Client.Resource(schema.GroupVersionResource{
		Group:    "batch",
		Version:  "v1",
		Resource: "jobs",
	}).Namespace(utd.GetNamespace()).Delete(context.TODO(), utd.GetName(), MetaV1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (this *Sysbench) SetHost(host string) *Sysbench {
	this.Host = host
	return this
}

func (this *Sysbench) InitDatabase() error {
	db, err := sql.Open("mysql", fmt.Sprintf("root:pgzxtppml@tcp(%s:3306)/", this.Host))
	if err != nil {
		fmt.Println(err)
	}
	defer db.Close()
	_, err = db.Exec("Drop DATABASE sysbench_test")
	if err != nil {
	}
	_, err = db.Exec("CREATE DATABASE sysbench_test")
	if err != nil {
		return err
	}
	return nil
}

func (this *Sysbench) IsComplete() bool {
	jobs, err := client.ClientSet.BatchV1().Jobs("default").Get(context.TODO(), this.Name, MetaV1.GetOptions{})
	if err != nil {

		return false
	}

	for _, jobConditions := range jobs.Status.Conditions {

		if jobConditions.Status == CoreV1.ConditionTrue && jobConditions.Type == BatchV1.JobComplete {
			return true
		}
	}
	return false
}

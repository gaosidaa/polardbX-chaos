package pdbx

import (
	"context"
	"fmt"
	PolarDBV1 "github.com/alibaba/polardbx-operator/api/v1"
	"github.com/alibaba/polardbx-operator/api/v1/common"
	"github.com/alibaba/polardbx-operator/api/v1/polardbx"
	CoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	MetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"log"
)

type PolarDB struct {
	Name   string
	Client dynamic.Interface
}

func NewPolarDB(name string) *PolarDB {
	return &PolarDB{Name: name}
}

func (this *PolarDB) InitOjb() *PolarDBV1.PolarDBXCluster {
	HostNetworkOk := true
	PolarDBClusterObj := PolarDBV1.PolarDBXCluster{
		TypeMeta: MetaV1.TypeMeta{
			APIVersion: "polardbx.aliyun.com/v1",
			Kind:       "PolarDBXCluster",
		},
		ObjectMeta: MetaV1.ObjectMeta{
			Name: this.Name,
		},
		Spec: PolarDBV1.PolarDBXClusterSpec{
			UpgradeStrategy: polardbx.RollingUpgradeStrategy,
			Config: polardbx.Config{
				DN: polardbx.DNConfig{
					MycnfOverwrite: `
print_gtid_info_during_recovery=1
gtid_mode = ON
enforce-gtid-consistency = 1
recovery_apply_binlog=on
slave_exec_mode=SMART`,
				},
			},
			Topology: polardbx.Topology{
				Nodes: polardbx.TopologyNodes{
					CDC: &polardbx.TopologyNodeCDC{
						Replicas: 2,
						Template: polardbx.CDCTemplate{
							Resources: CoreV1.ResourceRequirements{
								Limits: CoreV1.ResourceList{
									CoreV1.ResourceCPU:    resource.MustParse("3"),
									CoreV1.ResourceMemory: resource.MustParse("3Gi"),
								},
								Requests: CoreV1.ResourceList{
									CoreV1.ResourceCPU:    resource.MustParse("100m"),
									CoreV1.ResourceMemory: resource.MustParse("500Mi"),
								},
							},
						},
					},
					CN: polardbx.TopologyNodeCN{
						Replicas: 2,
						Template: polardbx.CNTemplate{
							Resources: CoreV1.ResourceRequirements{
								Limits: CoreV1.ResourceList{
									CoreV1.ResourceCPU:    resource.MustParse("5"),
									CoreV1.ResourceMemory: resource.MustParse("4Gi"),
								},
								Requests: CoreV1.ResourceList{
									CoreV1.ResourceCPU:    resource.MustParse("100m"),
									CoreV1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
					DN: polardbx.TopologyNodeDN{
						Replicas: 2,
						Template: polardbx.XStoreTemplate{
							Engine:      "galaxy",
							HostNetwork: &HostNetworkOk,
							Resources: common.ExtendedResourceRequirements{
								ResourceRequirements: CoreV1.ResourceRequirements{
									Limits: CoreV1.ResourceList{
										CoreV1.ResourceCPU:    resource.MustParse("3"),
										CoreV1.ResourceMemory: resource.MustParse("4Gi"),
									},
									Requests: CoreV1.ResourceList{
										CoreV1.ResourceCPU:    resource.MustParse("100m"),
										CoreV1.ResourceMemory: resource.MustParse("500Mi"),
									},
								},
							},
						},
					},
					GMS: polardbx.TopologyNodeGMS{
						Template: &polardbx.XStoreTemplate{
							Resources: common.ExtendedResourceRequirements{
								ResourceRequirements: CoreV1.ResourceRequirements{
									Limits: CoreV1.ResourceList{
										CoreV1.ResourceCPU:    resource.MustParse("2"),
										CoreV1.ResourceMemory: resource.MustParse("2Gi"),
									},
									Requests: CoreV1.ResourceList{
										CoreV1.ResourceCPU:    resource.MustParse("100m"),
										CoreV1.ResourceMemory: resource.MustParse("500Mi"),
									},
								},
							},
							ServiceType: CoreV1.ServiceTypeClusterIP,
							Engine:      "galaxy",
							HostNetwork: &HostNetworkOk,
						},
					},
				},
			},
		},
	}

	return &PolarDBClusterObj
}

func (this *PolarDB) CreatePolarDB() {

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(this.InitOjb())
	utd := unstructured.Unstructured{
		Object: obj,
	}
	if err != nil {
		log.Fatalln(err)
	}
	_, err = this.Client.Resource(schema.GroupVersionResource{
		Group:    "polardbx.aliyun.com",
		Version:  "v1",
		Resource: "polardbxclusters",
	}).Namespace("default").Create(context.TODO(), &utd, MetaV1.CreateOptions{})
	if err != nil {
		log.Fatalln(err)
	}
}

func (this *PolarDB) DeletePolarDB() {
	err := this.Client.Resource(schema.GroupVersionResource{
		Group:    "polardbx.aliyun.com",
		Version:  "v1",
		Resource: "polardbxclusters",
	}).Namespace("default").Delete(context.TODO(), this.Name, MetaV1.DeleteOptions{})
	if err != nil {
		fmt.Println(err)
		log.Fatalln(err)
	}
}

func (this *PolarDB) InjectDynamicClient(client dynamic.Interface) *PolarDB {
	this.Client = client
	return this
}

func (this *PolarDB) GetStatus() {

}

package chaos

import (
	"ChaosApi/pkg/client"
	"context"
	"encoding/json"
	"fmt"
	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	CoreV1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	MetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"log"
)

const (
	WorkflowAPIVersion = "chaos-mesh.org/v1alpha1"
	WorkflowKind       = "Workflow"
	WorkflowGroup      = "chaos-mesh.org"
	WorkflowVersion    = "v1alpha1"
	WorkflowResource   = "workflows"
)

type WorkflowBase struct {
	Name                 string `json:"name"`
	Namespace            string `json:"namespace"`
	Entry                string `json:"entry"`
	WorkflowChildrenName string
	ChildrenDeadline     string
	MasterDeadline       string
	Client               dynamic.Interface
	Templates            []v1alpha1.Template `json:"templates"`
}

//注入原子故障
func (this *WorkflowBase) InjectTemplate(templates ...v1alpha1.Template) *WorkflowBase {
	this.Templates = append(this.Templates, templates...)
	return this
}

// 设置类型 并且生产父、子工作流
func (this *WorkflowBase) SetWorkflowType(Type v1alpha1.TemplateType, masterDeadline, childrenDeadline string) *WorkflowBase {

	// 设置Chaos的注入类型
	// Serial 串行 ; Parallel  并行;
	workflowChildrenTemplate, workflowMasterTemplate := NewChaosTemplate(this.WorkflowChildrenName), NewChaosTemplate("entry")

	workflowChildrenTemplate.Deadline = &this.ChildrenDeadline
	workflowChildrenTemplate.Type = Type
	workflowChildrenTemplate.Deadline = &childrenDeadline

	workflowMasterTemplate.Deadline = &this.MasterDeadline
	workflowMasterTemplate.Type = Type
	workflowMasterTemplate.Deadline = &masterDeadline

	// 将所有的原子故障节点 注入到workflow子节点
	for _, item := range this.Templates {
		workflowChildrenTemplate.Children = append(workflowChildrenTemplate.Children, item.Name)
	}
	this.Templates = append(this.Templates, workflowChildrenTemplate)

	// 父节点关联子节点
	workflowMasterTemplate.Children = append(workflowMasterTemplate.Children, this.WorkflowChildrenName)
	this.Templates = append(this.Templates, workflowMasterTemplate)

	return this
}

// 注入动态客户端
func (this *WorkflowBase) InjectDynamicClient(client dynamic.Interface) *WorkflowBase {
	this.Client = client
	return this
}

func (this *WorkflowBase) CreateWorkflow() error {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(NewWorkflow(this))
	if err != nil {
		log.Fatalln(err)
	}
	utd := unstructured.Unstructured{
		Object: obj,
	}
	this.DeleteWorkflow()

	oldUtd, err := this.Client.Resource(schema.GroupVersionResource{
		Group:    WorkflowGroup,
		Version:  WorkflowVersion,
		Resource: WorkflowResource,
	}).Namespace(utd.GetNamespace()).Get(context.TODO(), utd.GetName(), metav1.GetOptions{})

	if apierrors.IsNotFound(err) {
		_, err = client.DynamicClient.Resource(schema.GroupVersionResource{
			Group:    WorkflowGroup,
			Version:  WorkflowVersion,
			Resource: WorkflowResource,
		}).Namespace(utd.GetNamespace()).Create(context.TODO(), &utd, metav1.CreateOptions{})
	} else {
		return fmt.Errorf("%s  已经存在 ", oldUtd.GetName())
	}

	return nil
}

func (this *WorkflowBase) DeleteWorkflow() error {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(NewWorkflow(this))
	if err != nil {
		log.Fatalln(err)
	}
	utd := unstructured.Unstructured{
		Object: obj,
	}

	err = client.DynamicClient.Resource(schema.GroupVersionResource{
		Group:    WorkflowGroup,
		Version:  WorkflowVersion,
		Resource: WorkflowResource,
	}).Namespace(utd.GetNamespace()).Delete(context.TODO(), utd.GetName(), metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

func NewWorkflowBase(name, namespace, workflowChildrenName string) *WorkflowBase {
	return &WorkflowBase{
		Name:                 name,
		Namespace:            namespace,
		WorkflowChildrenName: workflowChildrenName,
	}
}
func NewChaosTemplate(name string) v1alpha1.Template {
	return v1alpha1.Template{Name: name}
}
func NewWorkflow(base *WorkflowBase) *v1alpha1.Workflow {
	return &v1alpha1.Workflow{
		TypeMeta: MetaV1.TypeMeta{
			APIVersion: WorkflowAPIVersion,
			Kind:       WorkflowKind,
		},
		ObjectMeta: MetaV1.ObjectMeta{
			Name:      base.Name,
			Namespace: base.Namespace,
		},
		Spec: v1alpha1.WorkflowSpec{
			Entry:     "entry",
			Templates: base.Templates,
		},
	}
}

func (this *WorkflowBase) GetStatus() bool {
	utd, err := this.Client.Resource(schema.GroupVersionResource{
		Group:    WorkflowGroup,
		Version:  WorkflowVersion,
		Resource: WorkflowResource,
	}).Namespace(this.Namespace).Get(context.TODO(), this.Name, metav1.GetOptions{})
	if err !=nil {
		fmt.Println(err)
		return false
	}
	utdByte ,err :=utd.MarshalJSON()
	if err !=nil {
		fmt.Println(err)
		return false
	}
	workflow :=&v1alpha1.Workflow{}
	err = json.Unmarshal(utdByte,workflow)
	if err !=nil {
		fmt.Println(err)
		return false
	}


	for _,item := range  workflow.Status.Conditions {
		//fmt.Println(item.Status )
		if item.Status == CoreV1.ConditionTrue && item.Type == v1alpha1.WorkflowConditionAccomplished {
			return  true
		}
	}
	return  false

}
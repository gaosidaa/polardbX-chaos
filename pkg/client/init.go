package client

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

var DynamicClient dynamic.Interface

// 初始化动态客户端
func InitDynamicClient(configPath string) (dynamic.Interface, error) {

	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, err
	}
	DynamicClient, err = dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return DynamicClient, nil
}

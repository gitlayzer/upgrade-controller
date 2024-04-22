package controller

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

// GetClientSet 获取一个 clientSet
func GetClientSet() *kubernetes.Clientset {
	// 从本地文件加载 kubeConfig
	conf, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	if err != nil {
		panic(err)
	}

	// 使用获取到的 kubeConfig 创建一个 clientSet
	clientSet, err := kubernetes.NewForConfig(conf)
	if err != nil {
		panic(err)
	}

	// 返回 clientSet
	return clientSet
}

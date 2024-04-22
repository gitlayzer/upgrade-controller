package utils

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// GetPodsByDeployment 用于根据 Deployment 获取关联的 Pod 列表
func GetPodsByDeployment(deploymentName, namespace string, client *kubernetes.Clientset) []corev1.Pod {
	deployment, err := client.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		klog.Error("Create ClientSet error: ", err)
		return nil
	}

	rsIds := getRsIdsByDeployment(deployment, client)
	podsList := make([]corev1.Pod, 0)
	for _, rsId := range rsIds {
		pods := getPodsByReplicaSet(&rsId, client, namespace)
		podsList = append(podsList, pods...)
	}

	return podsList
}

// getRsIdsByDeployment 用于将 Deployment 关联的 ReplicaSet 列表获取出来
func getRsIdsByDeployment(deployment *appsv1.Deployment, client *kubernetes.Clientset) []appsv1.ReplicaSet {
	rsList, err := client.AppsV1().ReplicaSets(deployment.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.Set(deployment.Spec.Selector.MatchLabels).String(),
	})
	if err != nil {
		klog.Error("List ReplicaSets error: ", err)
		return nil
	}

	rsIds := make([]appsv1.ReplicaSet, len(rsList.Items))

	for _, rs := range rsList.Items {
		rsIds = append(rsIds, rs)
	}
	return rsIds
}

// getPodsByReplicaSet 用于根据 ReplicaSet 获取关联的 Pod 列表
func getPodsByReplicaSet(replicaSet *appsv1.ReplicaSet, client *kubernetes.Clientset, namespace string) []corev1.Pod {
	podList, err := client.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Error("List Pods error: ", err)
		return nil
	}

	pods := make([]corev1.Pod, 0)
	for _, pod := range podList.Items {
		// 找到 pod OwnerReferences uid 相同的 pod
		if pod.OwnerReferences != nil && len(pod.OwnerReferences) == 1 && pod.OwnerReferences[0].UID == replicaSet.UID {
			pods = append(pods, pod)
		}
	}
	return pods
}

// GetDeploymentReplicas 用于获取 Deployment 的副本数
func GetDeploymentReplicas(deploymentName, namespace string, client *kubernetes.Clientset) int32 {
	deployment, err := client.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		klog.Error("Get Deployment error: ", err)
		return 0
	}

	return deployment.Status.Replicas
}

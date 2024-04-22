package controller

import (
	"context"
	"fmt"
	jsonpath "github.com/evanphx/json-patch"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type patchUpgradeOperation struct {
	Op   string `json:"op"`
	Path string `json:"path"`

	Value interface{} `json:"value,omitempty"`
}

// UpgradePodByImages 更新 Pod 的镜像
func UpgradePodByImages(ctx context.Context, pod *corev1.Pod, image []string, clientSet *kubernetes.Clientset) error {
	klog.Infof("Start to upgrade pod %s", pod.Name)
	patchList := make([]patchUpgradeOperation, 0)
	for k, imageName := range image {
		p := patchUpgradeOperation{
			Op:    "replace",
			Path:  fmt.Sprintf("/spec/containers/%v/image", k),
			Value: imageName,
		}
		patchList = append(patchList, p)
	}
	patchBytes, err := json.Marshal(patchList)
	if err != nil {
		klog.Error(err)
		return err
	}

	jsonPatch, err := jsonpath.DecodePatch(patchBytes)
	if err != nil {
		klog.Error("DecodePatch error: ", err)
		return err
	}

	jsonPatchBytes, err := json.Marshal(jsonPatch)
	if err != nil {
		klog.Error("json Marshal error: ", err)
		return err
	}

	_, err = clientSet.CoreV1().Pods(pod.Namespace).Patch(ctx, pod.Name, types.JSONPatchType, jsonPatchBytes, metav1.PatchOptions{})
	if err != nil {
		klog.Error("Patch pod error: ", err)
		return err
	}

	klog.Info("Upgrade pod success")

	return nil
}

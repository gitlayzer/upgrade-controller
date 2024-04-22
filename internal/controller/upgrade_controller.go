/*
Copyright 2024 gitlayzer.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"github.com/gitlayzer/upgrade-controller/internal/utils"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	devopsengineercomcnv1alpha1 "github.com/gitlayzer/upgrade-controller/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PodStatusSuccess = "Successful"
	PodStatusRunning = "Running"
	PodStatusFailed  = "Failure"

	Upgrade = "upgrade"
)

// UpGradeReconciler reconciles a UpGrade object
type UpGradeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=devops-engineer.com.cn,resources=upgrades,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=devops-engineer.com.cn,resources=upgrades/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=devops-engineer.com.cn,resources=upgrades/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the UpGrade object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.2/pkg/reconcile
func (r *UpGradeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// 打印控制器启动的日志
	_ = ctrl.Log.WithName("UpGrader").WithValues("UpGrader", req.NamespacedName)

	UpGrade := &devopsengineercomcnv1alpha1.UpGrade{}

	// 获取 UpGrade 对象
	err := r.Get(ctx, req.NamespacedName, UpGrade)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			klog.Error("Failed to get UpGrader: ", err)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// 如果 UpGrade 的 Status.Status 为 Successful，则直接返回不再执行后续操作
	if UpGrade.Status.Status == PodStatusSuccess {
		klog.Info("UpGrade pod is successful")
		return reconcile.Result{}, nil
	}

	// 如果 UpGrade 的 Spec.Type 为 upgrade，则将 UpGrade 的 Status.Type 设置为 upgrade
	if UpGrade.Spec.Type == Upgrade {
		klog.Info("UpGrade type is upgrade")
		UpGrade.Status.Type = Upgrade
	}

	// 如果 UpGrade 的 Status.Status 为 Running，则直接返回不再执行后续操作
	UpGrade.Status.Status = PodStatusRunning
	err = r.Client.Status().Update(ctx, UpGrade)
	if err != nil {
		klog.Error("Failed to update UpGrade status: ", err)
		return reconcile.Result{}, err
	}

	// 获取 Deployment 的 pod 列表
	podList := utils.GetPodsByDeployment(UpGrade.Spec.DeploymentRef.Name, UpGrade.Spec.DeploymentRef.Namespace, GetClientSet())
	if len(podList) == 0 {
		klog.Error("Failed to get pod list")
		return reconcile.Result{}, nil
	}

	// 获取升级副本数
	var replicaCount = UpGrade.Spec.UpgradeReplicas

	// 如果 UpgradeReplicas 指定为 0，则使用 pod 数量作为升级副本数
	if UpGrade.Spec.UpgradeReplicas == 0 {
		// 则去获取 Deployment 的副本数
		utils.GetDeploymentReplicas(UpGrade.Spec.DeploymentRef.Name, UpGrade.Spec.DeploymentRef.Namespace, GetClientSet())

		// 将 Deployment 的副本数作为升级副本数
		UpGrade.Spec.UpgradeReplicas = replicaCount
	}

	// 如果升级副本数大于 pod 数量，则将升级副本数设置为 pod 数量
	if len(podList) < UpGrade.Spec.UpgradeReplicas {
		replicaCount = len(podList)
	}

	// 获取升级镜像列表
	imageList := make([]string, 0)
	if UpGrade.Spec.Type == Upgrade {
		for _, v := range UpGrade.Spec.Images {
			imageList = append(imageList, v.Image)
		}
	}

	// 升级 pod
	for i := 0; i < replicaCount; i++ {
		err = UpgradePodByImages(ctx, &podList[i], imageList, GetClientSet())
		if err != nil {
			klog.Error("Failed to upgrade pod: ", err)
			UpGrade.Status.Status = PodStatusFailed
			err = r.Client.Status().Update(ctx, UpGrade)
			if err != nil {
				klog.Error("Failed to update UpGrader status: ", err)
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: false}, err
		}
	}

	// 升级成功, 更新 UpGrader 的 Status
	UpGrade.Status.Status = PodStatusSuccess
	err = r.Client.Status().Update(ctx, UpGrade)
	if err != nil {
		klog.Error("Failed to update UpGrade status: ", err)
		return reconcile.Result{}, err
	}

	// 将更新的副本数写入到 UpGrader 的 Status 中
	UpGrade.Status.UpGradeReplicas = replicaCount
	err = r.Client.Status().Update(ctx, UpGrade)
	if err != nil {
		klog.Error("Failed to update UpGrade status: ", err)
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UpGradeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&devopsengineercomcnv1alpha1.UpGrade{}).
		Complete(r)
}

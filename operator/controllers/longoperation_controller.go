/*
Copyright 2022.

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

package controllers

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	operatorApi "github.com/Tomasz-Smelcerz-SAP/kyma-operator-random/k8s-api/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// LongOperationReconciler reconciles a LongOperation object
type LongOperationReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	RandomGen *rand.Rand
}

const timeFormat = "2006-01-02T15:04:05.999Z07:00"

//+kubebuilder:rbac:groups=kyma.kyma-project.io,resources=longoperations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kyma.kyma-project.io,resources=longoperations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kyma.kyma-project.io,resources=longoperations/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the LongOperation object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *LongOperationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling:", "object", req.String())
	obj := operatorApi.LongOperation{}
	err := r.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &obj)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info(req.NamespacedName.String() + " got deleted!")
		}
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	constantTime := obj.Spec.ConstantProcessingTime
	randomTime := obj.Spec.RandomProcessingTime

	st := obj.Status.State
	if st == "" {
		if constantTime <= 0 && randomTime <= 0 {
			logger.Info(fmt.Sprintf("LongOperation %s has invalid time configuration", req.NamespacedName.String()))
			obj.Status.State = operatorApi.LongOperationStateError
			err = r.Status().Update(ctx, &obj)
			if err != nil {
				logger.Error(err, fmt.Sprintf("Error during status update of LongOperation %s", req.NamespacedName.String()))
			}
			return ctrl.Result{}, nil
		}

		processingTime := r.calcProcessingTime(constantTime, randomTime)

		logger.Info(fmt.Sprintf("LongOperation %s will be processed in %d seconds", req.NamespacedName.String(), processingTime))

		obj.Status.State = operatorApi.LongOperationStateProcessing
		targetTime := time.Now().Add(time.Duration(processingTime-1) * time.Second)
		obj.Status.RecheckAfter = targetTime.Format(timeFormat)
		err = r.Status().Update(ctx, &obj)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Error during status update of LongOperation %s", req.NamespacedName.String()))
		}

		return ctrl.Result{RequeueAfter: time.Duration(processingTime) * time.Second}, nil
	}

	if st == operatorApi.LongOperationStateProcessing {
		if obj.Status.RecheckAfter != "" {
			targetTime, err := time.Parse(timeFormat, obj.Status.RecheckAfter)
			if err != nil {
				logger.Error(err, fmt.Sprintf("Skipping processing of LongOperation %s - error parsing time", req.NamespacedName.String()))
				return ctrl.Result{}, nil
			}

			if targetTime.After(time.Now()) {
				//Skip processing, time has not passed yet
				logger.Info(fmt.Sprintf("Skipping processing of LongOperation %s - not enough time has passed yet", req.NamespacedName.String()))
				return ctrl.Result{}, nil
			}
		}

		logger.Info(fmt.Sprintf("LongOperation %s has finished processing", req.NamespacedName.String()))
		obj.Status.State = operatorApi.LongOperationStateReady
		err = r.Status().Update(ctx, &obj)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Error during status update of LongOperation %s", req.NamespacedName.String()))
		}
		return ctrl.Result{RequeueAfter: time.Second * 60}, nil
	}

	if st == operatorApi.LongOperationStateReady {
		logger.Info(fmt.Sprintf("LongOperation %s is already processed, no action taken", req.NamespacedName.String()))
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LongOperationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&operatorApi.LongOperation{}).
		Complete(r)
}

func (r *LongOperationReconciler) calcProcessingTime(constantTime, randomTime int) int {
	if randomTime == 0 {
		return constantTime
	}
	res := constantTime + r.RandomGen.Intn(randomTime+1)
	if res < 1 {
		res = 1
	}

	return res
}

//t.Format("2006-01-02T15:04:05.999Z07:00")

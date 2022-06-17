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
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	operatorApi "github.com/Tomasz-Smelcerz-SAP/kyma-operator-random/k8s-api/api/v1alpha1"
	"github.com/go-logr/logr"
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

	updateStatusFn, err := r.process(&obj, logger)
	if err != nil {
		obj.Status.State = operatorApi.LongOperationStateError
		obj.Status.Message = err.Error()
		err = r.Status().Update(ctx, &obj)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Error during status update of LongOperation %s", req.NamespacedName.String()))
		}
		return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	}

	statusChanged := updateStatusFn(&obj.Status)
	if obj.Status.ObservedGeneration != int(obj.Generation) {
		obj.Status.ObservedGeneration = int(obj.Generation)
		statusChanged = true
	}

	if statusChanged {
		obj.Status.Updated = time.Now().Format(timeFormat)
		err = r.Status().Update(ctx, &obj)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Error during status update of LongOperation %s", req.NamespacedName.String()))
			return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
		}
	}

	if obj.Status.State == operatorApi.LongOperationStateProcessing {
		return ctrl.Result{RequeueAfter: 20 * time.Second}, nil
	}
	if obj.Status.State == operatorApi.LongOperationStateReady {
		return ctrl.Result{RequeueAfter: 300 * time.Second}, nil
	}

	return ctrl.Result{RequeueAfter: 600 * time.Second}, nil

}

func (r *LongOperationReconciler) verifySpec(obj *operatorApi.LongOperation) error {
	constantTime := obj.Spec.ConstantProcessingTime
	randomTime := obj.Spec.RandomProcessingTime

	if constantTime <= 0 && randomTime <= 0 {
		msg := fmt.Sprintf("LongOperation %s/%s has invalid time configuration", obj.Namespace, obj.Name)
		return errors.New(msg)
	}

	return nil
}

func (r *LongOperationReconciler) compareDesiredStateWithCurrent(obj *operatorApi.LongOperation, logger logr.Logger) (*differences, error) {
	diffs := differences{}

	now := time.Now()

	if obj.Status.State == "" || obj.Status.BusyUntil == "" {
		processingTime := r.calcProcessingTime(string(obj.UID), obj.Spec.ConstantProcessingTime, obj.Spec.RandomProcessingTime)
		busyUntilTime := now.Add(time.Duration(processingTime) * time.Second)
		diffs.setBusyUntilTo = busyUntilTime.Format(timeFormat)
		diffs.changeStatusTo = operatorApi.LongOperationStateProcessing
		diffs.setMessageTo = fmt.Sprintf("Processing time: %d[s]", processingTime)
	} else {
		if int64(obj.Status.ObservedGeneration) == obj.Generation {

			//handle reconciliations without Spec change
			busyUntilTime, err := time.Parse(timeFormat, obj.Status.BusyUntil)
			if err != nil {
				msg := fmt.Sprintf("Error for LongOperation %s/%s: cannot parse BusyUntil time: %s", obj.Namespace, obj.Name, obj.Status.BusyUntil)
				logger.Error(err, msg)
				return nil, errors.New(msg)
			}

			if now.After(busyUntilTime) {
				//no longer busy!
				if obj.Status.State == operatorApi.LongOperationStateReady {
					diffs.none = true
				} else {
					diffs.changeStatusTo = operatorApi.LongOperationStateReady
				}
			} else {
				//still busy...
				if obj.Status.State == operatorApi.LongOperationStateProcessing {
					diffs.none = true
				} else {
					diffs.changeStatusTo = operatorApi.LongOperationStateProcessing
				}
			}
		} else {
			//handle reconciliations with a change to Spec
			processingTime := r.calcProcessingTime(string(obj.UID), obj.Spec.ConstantProcessingTime, obj.Spec.RandomProcessingTime)
			busyUntilTime := now.Add(time.Duration(processingTime) * time.Second)
			diffs.setBusyUntilTo = busyUntilTime.Format(timeFormat)
			diffs.changeStatusTo = operatorApi.LongOperationStateProcessing
			diffs.setMessageTo = fmt.Sprintf("Processing time: %d[s]", processingTime)
		}
	}

	return &diffs, nil
}

func (r *LongOperationReconciler) change(obj *operatorApi.LongOperation, diffs *differences, logger logr.Logger) (statusSetter, error) {
	return diffs.changeStatus(logger), nil
}

func (r *LongOperationReconciler) process(obj *operatorApi.LongOperation, logger logr.Logger) (statusSetter, error) {

	err := r.verifySpec(obj)
	if err != nil {
		return nil, err
	}

	differences, err := r.compareDesiredStateWithCurrent(obj, logger)
	if err != nil {
		return nil, err
	}

	if differences.None() {
		noopSetter := func(target *operatorApi.LongOperationStatus) bool {
			logger.Info("No changes detected - reconciliation is done.")
			return false
		}
		return noopSetter, nil
	}

	return r.change(obj, differences, logger)
}

// SetupWithManager sets up the controller with the Manager.
func (r *LongOperationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&operatorApi.LongOperation{}).
		Complete(r)
}

func (r *LongOperationReconciler) calcProcessingTime(uid string, constantTime, randomTimeFactor int) int {

	var res big.Int
	var divident big.Int
	var divisor = big.NewInt(int64(randomTimeFactor))

	divident.SetString(strings.Replace(uid, "-", "", 4), 16)
	res.Mod(&divident, divisor)
	randomTime := int(res.Int64())

	sum := constantTime + randomTime
	if sum < 1 {
		sum = 1
	}

	return sum
}

//differences Captures differences between desired state and the current state
type differences struct {
	none           bool
	setBusyUntilTo string
	setMessageTo   string
	changeStatusTo operatorApi.LongOperationState
}

func (d *differences) None() bool {
	return d.none
}

func (d *differences) changeStatus(logger logr.Logger) statusSetter {
	return func(target *operatorApi.LongOperationStatus) bool {
		changed := false

		if d.setBusyUntilTo != "" {
			logger.Info("Setting BusyUntil to: " + d.setBusyUntilTo)
			target.BusyUntil = d.setBusyUntilTo
			changed = true
		}
		if d.setMessageTo != "" {
			logger.Info("Setting message to: " + d.setMessageTo)
			target.Message = d.setMessageTo
			changed = true
		}
		if string(d.changeStatusTo) != "" {
			logger.Info("Setting state to: " + string(d.changeStatusTo))
			target.State = d.changeStatusTo
			changed = true
		}
		return changed
	}
}

type statusSetter func(status *operatorApi.LongOperationStatus) bool

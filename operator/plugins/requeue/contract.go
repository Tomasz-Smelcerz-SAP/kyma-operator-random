package requeue

import (
	"time"

	operatorAPI "github.com/Tomasz-Smelcerz-SAP/kyma-operator-random/k8s-api/api/v1alpha1"
)

//Contract for plugins that control Re-queueing

type RequeueDecision struct {
	RequeueAfter time.Duration
}

type Contract interface {
	GetRequeueDecision(apiObject *operatorAPI.LongOperation) (*RequeueDecision, error)
}

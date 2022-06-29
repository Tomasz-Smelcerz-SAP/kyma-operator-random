package main

import (
	"fmt"
	"math/rand"
	"time"

	operatorAPI "github.com/Tomasz-Smelcerz-SAP/kyma-operator-random/k8s-api/api/v1alpha1"
	pluginAPI "github.com/Tomasz-Smelcerz-SAP/kyma-operator-random/operator/plugins/requeue"
)

type Plugin struct {
}

func (p *Plugin) GetRequeueDecision(apiObject *operatorAPI.LongOperation) (*pluginAPI.RequeueDecision, error) {
	fmt.Println("Hello there from the plugin!")

	var seconds int = 45

	if apiObject.Status.State == operatorAPI.LongOperationStateProcessing {
		seconds = 20
	} else if apiObject.Status.State == operatorAPI.LongOperationStateReady {
		seconds = 180
	}

	return &pluginAPI.RequeueDecision{
		RequeueAfter: time.Duration(randomizeByTenPercent(seconds)) * time.Second,
	}, nil

}

func randomizeByTenPercent(val int) int {
	if val < 10 {
		return randomizeByTenPercent(10)
	}

	var tenPercentOfVal float64 = 0.1 * float64(val)
	delta := (1 - r.Float64()*2) * tenPercentOfVal //ranges from [-1*tenPercentOfVal ... 0 ... +1*tenPercentOfVal)
	return val + int(delta)
}

var (
	r               = rand.New(rand.NewSource(time.Now().UnixNano()))
	Exported Plugin = Plugin{}
)

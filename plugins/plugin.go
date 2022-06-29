package main

import (
	"fmt"

	operatorAPI "github.com/Tomasz-Smelcerz-SAP/kyma-operator-random/k8s-api/api/v1alpha1"
	pluginAPI "github.com/Tomasz-Smelcerz-SAP/kyma-operator-random/operator/plugins/requeue"
)

type Plugin struct {
}

func (p *Plugin) GetRequeueDecision(apiObject *operatorAPI.LongOperation) (*pluginAPI.RequeueDecision, error) {
	fmt.Println("Hello there from the plugin!")
	return nil, nil
}

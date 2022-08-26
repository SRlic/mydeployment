package utils

import (
	"fmt"

	mydeployments "mydeployment/api/v1"

	"github.com/go-logr/logr"
)

func StartLog(myDepolyment *mydeployments.MyDeployment, logr logr.Logger) {
	logr.Info("Reconcile start =====================")
}

func EndLog(myDepolyment *mydeployments.MyDeployment, logr logr.Logger) {
	myDepolyment.Status.Phase = GetStatusPhase(myDepolyment)
	logr.Info(fmt.Sprintf("My Depolyment state %v", myDepolyment.Status.Phase))
	logr.Info("=====================  Reconcile end")
}

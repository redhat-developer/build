package controller

import (
	"github.com/shipwright-io/build/pkg/controller/buildstrategy"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, buildstrategy.Add)
}

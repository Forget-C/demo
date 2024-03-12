package inplaceupdate

import v1 "github.com/Forget-C/demo/inplaceupdate/program/api/v1"

func IsCompleted(obj *v1.InplaceUpdate) bool {
	switch obj.Status.Phase {
	case v1.InplaceUpdatePhaseFinished, v1.InplaceUpdatePhaseFailed:
		return true
	}
	return false
}

func IsRunning(obj *v1.InplaceUpdate) bool {
	return obj.Status.Phase == v1.InplaceUpdatePhaseRunning
}

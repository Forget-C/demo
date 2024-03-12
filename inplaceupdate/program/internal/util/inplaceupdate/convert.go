package inplaceupdate

import v1 "github.com/Forget-C/demo/inplaceupdate/program/api/v1"

func ContainerNames(spec v1.InplaceUpdateSpec) []string {
	names := make([]string, 0, len(spec.Containers))
	for _, target := range spec.Containers {
		names = append(names, target.Name)
	}
	return names
}

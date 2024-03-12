package util

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func FindContainer(name string, spec corev1.PodSpec) *corev1.Container {
	for i := range spec.Containers {
		v := &spec.Containers[i]
		if v.Name == name {
			return v
		}
	}
	return nil
}

func FindContainerStatus(name string, status []corev1.ContainerStatus) *corev1.ContainerStatus {
	for i := range status {
		v := &status[i]
		if v.Name == name {
			return v
		}
	}
	return nil
}

func GetLatestContainerStatusMap(status []corev1.ContainerStatus, names ...string) map[string]*corev1.ContainerStatus {
	m := make(map[string]*corev1.ContainerStatus)
	for i := range status {
		if len(names) > 0 {
			found := false
			for _, name := range names {
				if status[i].Name == name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		m[status[i].Name] = &status[i]
	}
	return m
}

func PodNames(pods []*corev1.Pod) string {
	names := make([]string, 0, len(pods))
	for _, pod := range pods {
		names = append(names, pod.Name)
	}
	return strings.Join(names, ",")
}

func FindContainers(names []string, spec corev1.PodSpec) map[string]*corev1.Container {
	containers := make(map[string]*corev1.Container)
	for _, name := range names {
		container := FindContainer(name, spec)
		if container != nil {
			containers[name] = container
		}
	}
	return containers
}

// ContainerMerge merge two container slices
// Update a using data from b
func ContainerMerge(a, b []corev1.Container) []corev1.Container {
	m := make(map[string]corev1.Container)
	for _, v := range a {
		m[v.Name] = v
	}
	for _, v := range b {
		m[v.Name] = v
	}
	result := make([]corev1.Container, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	return result
}

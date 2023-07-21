package servicetypes

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var solr = ServiceType{
	Name: "solr-php-persistent", // this has to be like this because it is used in selectors, and is unchangeable now on existing deployed solr
	Ports: ServicePorts{
		Ports: []corev1.ServicePort{
			{
				Port: 8983,
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 8983,
				},
				Protocol: corev1.ProtocolTCP,
				Name:     "tcp-8983",
			},
		},
	},
	PrimaryContainer: ServiceContainer{
		Name:            "solr",
		ImagePullPolicy: corev1.PullAlways,
		Container: corev1.Container{
			Ports: []corev1.ContainerPort{
				{
					Name:          "tcp-8983",
					ContainerPort: 8983,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 8983,
						},
					},
				},
				InitialDelaySeconds: 1,
				PeriodSeconds:       3,
			},
			LivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 8983,
						},
					},
				},
				InitialDelaySeconds: 90,
				TimeoutSeconds:      3,
				FailureThreshold:    5,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("10m"),
					corev1.ResourceMemory: resource.MustParse("100M"),
				},
			},
		},
	},
	Volumes: ServiceVolume{
		PersistentVolumeSize: "5Gi",
		PersistentVolumeType: corev1.ReadWriteOnce,
		PersistentVolumePath: "/var/solr",
		BackupConfiguration: BackupConfiguration{
			Command:       `/bin/sh -c 'tar -cf - -C "/var/solr" --exclude="lost\+found" . || [ $? -eq 1 ]'`,
			FileExtension: ".{{ .OverrideName }}.tar",
		},
	},
}

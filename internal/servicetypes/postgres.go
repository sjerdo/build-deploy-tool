package servicetypes

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var postgresSingle = ServiceType{
	Name: "postgres-single",
	Ports: ServicePorts{
		Ports: []corev1.ServicePort{
			{
				Port: 5432,
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 5432,
				},
				Protocol: corev1.ProtocolTCP,
				Name:     "5432-tcp",
			},
		},
	},
	PrimaryContainer: ServiceContainer{
		Name:            "postgres",
		ImagePullPolicy: corev1.PullAlways,
		Container: corev1.Container{
			Ports: []corev1.ContainerPort{
				{
					Name:          "5432-tcp",
					ContainerPort: 5432,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 5432,
						},
					},
				},
				InitialDelaySeconds: 1,
				TimeoutSeconds:      1,
			},
			LivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 5432,
						},
					},
				},
				InitialDelaySeconds: 120,
				PeriodSeconds:       5,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("10m"),
					corev1.ResourceMemory: resource.MustParse("100M"),
				},
			},
		},
	},
	PodSecurityContext: ServicePodSecurityContext{
		HasDefault: true,
		FSGroup:    0,
	},
	Strategy: appsv1.DeploymentStrategy{
		Type: appsv1.RecreateDeploymentStrategyType,
	},
	Volumes: ServiceVolume{
		PersistentVolumeSize: "5Gi",
		PersistentVolumeType: corev1.ReadWriteOnce,
		PersistentVolumePath: "/var/lib/postgresql/data",
		BackupConfiguration: BackupConfiguration{
			Command:       `/bin/sh -c "PGPASSWORD=${{ .OverrideName | FixServiceName }}_PASSWORD pg_dump --host=localhost --port=${{ .OverrideName | FixServiceName }}_SERVICE_PORT --dbname=${{ .OverrideName | FixServiceName }}_DB --username=${{ .OverrideName | FixServiceName }}_USER --format=t -w"`,
			FileExtension: ".{{ .OverrideName }}.tar",
		},
	},
}

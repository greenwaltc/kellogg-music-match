package main

import (
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	networkingv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create the namespace
		namespace, err := corev1.NewNamespace(ctx, "kellogg-music-match", &corev1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("kellogg-music-match"),
				Labels: pulumi.StringMap{
					"app": pulumi.String("kellogg-music-match"),
				},
			},
		})
		if err != nil {
			return err
		}

		// Create the service account
		serviceAccount, err := corev1.NewServiceAccount(ctx, "kellogg-music-match-sa", &corev1.ServiceAccountArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("kellogg-music-match"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app": pulumi.String("kellogg-music-match"),
				},
			},
		})
		if err != nil {
			return err
		}

		// Create ConfigMap for UI configuration
		uiConfigMap, err := corev1.NewConfigMap(ctx, "ui-config", &corev1.ConfigMapArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("kellogg-music-match-ui-config"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kellogg-music-match"),
					"component": pulumi.String("ui"),
				},
			},
			Data: pulumi.StringMap{
				"config.json": pulumi.String("{\"apiBaseUrl\": \"http://kmm-backend.traefik.me\"}"),
			},
		})
		if err != nil {
			return err
		}

		// Create backend deployment
		backendDeployment, err := appsv1.NewDeployment(ctx, "backend-deployment", &appsv1.DeploymentArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("kellogg-music-match-backend"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kellogg-music-match"),
					"component": pulumi.String("backend"),
				},
			},
			Spec: &appsv1.DeploymentSpecArgs{
				Replicas: pulumi.Int(2),
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: pulumi.StringMap{
						"app":       pulumi.String("kellogg-music-match"),
						"component": pulumi.String("backend"),
					},
				},
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Labels: pulumi.StringMap{
							"app":       pulumi.String("kellogg-music-match"),
							"component": pulumi.String("backend"),
						},
					},
					Spec: &corev1.PodSpecArgs{
						ServiceAccountName: serviceAccount.Metadata.Name(),
						Containers: corev1.ContainerArray{
							&corev1.ContainerArgs{
								Name:            pulumi.String("backend"),
								Image:           pulumi.String("kellogg-music-match-backend:latest"),
								ImagePullPolicy: pulumi.String("Never"),
								Ports: corev1.ContainerPortArray{
									&corev1.ContainerPortArgs{
										ContainerPort: pulumi.Int(8080),
										Name:          pulumi.String("http"),
									},
								},
								Env: corev1.EnvVarArray{
									&corev1.EnvVarArgs{
										Name:  pulumi.String("PORT"),
										Value: pulumi.String("8080"),
									},
								},
								Resources: &corev1.ResourceRequirementsArgs{
									Requests: pulumi.StringMap{
										"cpu":    pulumi.String("100m"),
										"memory": pulumi.String("128Mi"),
									},
									Limits: pulumi.StringMap{
										"cpu":    pulumi.String("500m"),
										"memory": pulumi.String("512Mi"),
									},
								},
								LivenessProbe: &corev1.ProbeArgs{
									HttpGet: &corev1.HTTPGetActionArgs{
										Path: pulumi.String("/health"),
										Port: pulumi.String("http"),
									},
									InitialDelaySeconds: pulumi.Int(30),
									PeriodSeconds:       pulumi.Int(10),
								},
								ReadinessProbe: &corev1.ProbeArgs{
									HttpGet: &corev1.HTTPGetActionArgs{
										Path: pulumi.String("/health"),
										Port: pulumi.String("http"),
									},
									InitialDelaySeconds: pulumi.Int(5),
									PeriodSeconds:       pulumi.Int(5),
								},
							},
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}

		// Create backend service
		backendService, err := corev1.NewService(ctx, "backend-service", &corev1.ServiceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("kellogg-music-match-backend"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kellogg-music-match"),
					"component": pulumi.String("backend"),
				},
			},
			Spec: &corev1.ServiceSpecArgs{
				Type: pulumi.String("ClusterIP"),
				Ports: corev1.ServicePortArray{
					&corev1.ServicePortArgs{
						Name:       pulumi.String("http"),
						Port:       pulumi.Int(8080),
						TargetPort: pulumi.String("http"),
						Protocol:   pulumi.String("TCP"),
					},
				},
				Selector: pulumi.StringMap{
					"app":       pulumi.String("kellogg-music-match"),
					"component": pulumi.String("backend"),
				},
			},
		})
		if err != nil {
			return err
		}

		// Create UI deployment
		uiDeployment, err := appsv1.NewDeployment(ctx, "ui-deployment", &appsv1.DeploymentArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("kellogg-music-match-ui"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kellogg-music-match"),
					"component": pulumi.String("ui"),
				},
			},
			Spec: &appsv1.DeploymentSpecArgs{
				Replicas: pulumi.Int(2),
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: pulumi.StringMap{
						"app":       pulumi.String("kellogg-music-match"),
						"component": pulumi.String("ui"),
					},
				},
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Labels: pulumi.StringMap{
							"app":       pulumi.String("kellogg-music-match"),
							"component": pulumi.String("ui"),
						},
					},
					Spec: &corev1.PodSpecArgs{
						ServiceAccountName: serviceAccount.Metadata.Name(),
						Containers: corev1.ContainerArray{
							&corev1.ContainerArgs{
								Name:            pulumi.String("ui"),
								Image:           pulumi.String("kellogg-music-match-ui:latest"),
								ImagePullPolicy: pulumi.String("Never"),
								Ports: corev1.ContainerPortArray{
									&corev1.ContainerPortArgs{
										ContainerPort: pulumi.Int(80),
										Name:          pulumi.String("http"),
									},
								},
								Resources: &corev1.ResourceRequirementsArgs{
									Requests: pulumi.StringMap{
										"cpu":    pulumi.String("50m"),
										"memory": pulumi.String("64Mi"),
									},
									Limits: pulumi.StringMap{
										"cpu":    pulumi.String("200m"),
										"memory": pulumi.String("256Mi"),
									},
								},
								LivenessProbe: &corev1.ProbeArgs{
									HttpGet: &corev1.HTTPGetActionArgs{
										Path: pulumi.String("/"),
										Port: pulumi.String("http"),
									},
									InitialDelaySeconds: pulumi.Int(30),
									PeriodSeconds:       pulumi.Int(10),
								},
								ReadinessProbe: &corev1.ProbeArgs{
									HttpGet: &corev1.HTTPGetActionArgs{
										Path: pulumi.String("/"),
										Port: pulumi.String("http"),
									},
									InitialDelaySeconds: pulumi.Int(5),
									PeriodSeconds:       pulumi.Int(5),
								},
								VolumeMounts: corev1.VolumeMountArray{
									&corev1.VolumeMountArgs{
										Name:      pulumi.String("config"),
										MountPath: pulumi.String("/usr/share/nginx/html/config.json"),
										SubPath:   pulumi.String("config.json"),
									},
								},
							},
						},
						Volumes: corev1.VolumeArray{
							&corev1.VolumeArgs{
								Name: pulumi.String("config"),
								ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
									Name: uiConfigMap.Metadata.Name(),
								},
							},
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}

		// Create UI service
		uiService, err := corev1.NewService(ctx, "ui-service", &corev1.ServiceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("kellogg-music-match-ui"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kellogg-music-match"),
					"component": pulumi.String("ui"),
				},
			},
			Spec: &corev1.ServiceSpecArgs{
				Type: pulumi.String("ClusterIP"),
				Ports: corev1.ServicePortArray{
					&corev1.ServicePortArgs{
						Name:       pulumi.String("http"),
						Port:       pulumi.Int(80),
						TargetPort: pulumi.String("http"),
						Protocol:   pulumi.String("TCP"),
					},
				},
				Selector: pulumi.StringMap{
					"app":       pulumi.String("kellogg-music-match"),
					"component": pulumi.String("ui"),
				},
			},
		})
		if err != nil {
			return err
		}

		// Create ingress for both UI and backend
		ingress, err := networkingv1.NewIngress(ctx, "kellogg-music-match-ingress", &networkingv1.IngressArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("kellogg-music-match"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app": pulumi.String("kellogg-music-match"),
				},
				Annotations: pulumi.StringMap{
					"nginx.ingress.kubernetes.io/rewrite-target": pulumi.String("/"),
					"nginx.ingress.kubernetes.io/ssl-redirect":   pulumi.String("false"),
				},
			},
			Spec: &networkingv1.IngressSpecArgs{
				IngressClassName: pulumi.String("nginx"),
				Rules: networkingv1.IngressRuleArray{
					// UI ingress rule
					&networkingv1.IngressRuleArgs{
						Host: pulumi.String("kmm-ui.traefik.me"),
						Http: &networkingv1.HTTPIngressRuleValueArgs{
							Paths: networkingv1.HTTPIngressPathArray{
								&networkingv1.HTTPIngressPathArgs{
									Path:     pulumi.String("/"),
									PathType: pulumi.String("Prefix"),
									Backend: &networkingv1.IngressBackendArgs{
										Service: &networkingv1.IngressServiceBackendArgs{
											Name: pulumi.String("kellogg-music-match-ui"),
											Port: &networkingv1.ServiceBackendPortArgs{
												Number: pulumi.Int(80),
											},
										},
									},
								},
							},
						},
					},
					// Backend ingress rule
					&networkingv1.IngressRuleArgs{
						Host: pulumi.String("kmm-backend.traefik.me"),
						Http: &networkingv1.HTTPIngressRuleValueArgs{
							Paths: networkingv1.HTTPIngressPathArray{
								&networkingv1.HTTPIngressPathArgs{
									Path:     pulumi.String("/"),
									PathType: pulumi.String("Prefix"),
									Backend: &networkingv1.IngressBackendArgs{
										Service: &networkingv1.IngressServiceBackendArgs{
											Name: pulumi.String("kellogg-music-match-backend"),
											Port: &networkingv1.ServiceBackendPortArgs{
												Number: pulumi.Int(8080),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}

		// Export useful information
		ctx.Export("namespaceName", namespace.Metadata.Name())
		ctx.Export("serviceAccountName", serviceAccount.Metadata.Name())
		ctx.Export("backendDeploymentName", backendDeployment.Metadata.Name())
		ctx.Export("backendServiceName", backendService.Metadata.Name())
		ctx.Export("uiDeploymentName", uiDeployment.Metadata.Name())
		ctx.Export("uiServiceName", uiService.Metadata.Name())
		ctx.Export("ingressName", ingress.Metadata.Name())

		// Export application URLs
		ctx.Export("uiUrl", pulumi.String("http://kmm-ui.traefik.me"))
		ctx.Export("backendUrl", pulumi.String("http://kmm-backend.traefik.me"))

		// Export ingress status (will show external IP when available)
		ctx.Export("ingressStatus", ingress.Status)

		return nil
	})
}

package main

import (
	"os"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	networkingv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Read database schema and init script files
		schemaContent, err := os.ReadFile("../DATABASE_SCHEMA.sql")
		if err != nil {
			return err
		}

		initScriptContent, err := os.ReadFile("../init-database.sh")
		if err != nil {
			return err
		}
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
									&corev1.EnvVarArgs{
										Name:  pulumi.String("DB_HOST"),
										Value: pulumi.String("postgres"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("DB_PORT"),
										Value: pulumi.String("5432"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("DB_USER"),
										Value: pulumi.String("kellogg_user"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("DB_PASSWORD"),
										Value: pulumi.String("kellogg_secure_pass_2024"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("DB_NAME"),
										Value: pulumi.String("kellogg_music_match"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("DATABASE_URL"),
										Value: pulumi.String("postgres://kellogg_user:kellogg_secure_pass_2024@postgres:5432/kellogg_music_match?sslmode=disable"),
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

		// Create PostgreSQL Secret
		pgSecret, err := corev1.NewSecret(ctx, "postgres-secret", &corev1.SecretArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("postgres-secret"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kellogg-music-match"),
					"component": pulumi.String("database"),
				},
			},
			StringData: pulumi.StringMap{
				"POSTGRES_USER":     pulumi.String("kellogg_user"),
				"POSTGRES_PASSWORD": pulumi.String("kellogg_secure_pass_2024"),
				"POSTGRES_DB":       pulumi.String("kellogg_music_match"),
			},
		})
		if err != nil {
			return err
		}

		// Create PostgreSQL ConfigMap with schema and init script
		pgConfigMap, err := corev1.NewConfigMap(ctx, "postgres-config", &corev1.ConfigMapArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("postgres-config"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kellogg-music-match"),
					"component": pulumi.String("database"),
				},
			},
			Data: pulumi.StringMap{
				"PGDATA":              pulumi.String("/var/lib/postgresql/data/pgdata"),
				"init-database.sh":    pulumi.String(string(initScriptContent)),
				"DATABASE_SCHEMA.sql": pulumi.String(string(schemaContent)),
			},
		})
		if err != nil {
			return err
		}

		// Create PostgreSQL StatefulSet
		pgStatefulSet, err := appsv1.NewStatefulSet(ctx, "postgres-statefulset", &appsv1.StatefulSetArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("postgres"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kellogg-music-match"),
					"component": pulumi.String("database"),
				},
			},
			Spec: &appsv1.StatefulSetSpecArgs{
				ServiceName: pulumi.String("postgres"),
				Replicas:    pulumi.Int(1),
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: pulumi.StringMap{
						"app":       pulumi.String("kellogg-music-match"),
						"component": pulumi.String("database"),
					},
				},
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Labels: pulumi.StringMap{
							"app":       pulumi.String("kellogg-music-match"),
							"component": pulumi.String("database"),
						},
					},
					Spec: &corev1.PodSpecArgs{
						ServiceAccountName: serviceAccount.Metadata.Name(),
						Containers: corev1.ContainerArray{
							&corev1.ContainerArgs{
								Name:  pulumi.String("postgres"),
								Image: pulumi.String("kellogg-music-match-postgres:latest"),
								Ports: corev1.ContainerPortArray{
									&corev1.ContainerPortArgs{
										ContainerPort: pulumi.Int(5432),
										Name:          pulumi.String("postgres"),
									},
								},
								EnvFrom: corev1.EnvFromSourceArray{
									&corev1.EnvFromSourceArgs{
										SecretRef: &corev1.SecretEnvSourceArgs{
											Name: pgSecret.Metadata.Name(),
										},
									},
									&corev1.EnvFromSourceArgs{
										ConfigMapRef: &corev1.ConfigMapEnvSourceArgs{
											Name: pgConfigMap.Metadata.Name(),
										},
									},
								},
								VolumeMounts: corev1.VolumeMountArray{
									&corev1.VolumeMountArgs{
										Name:      pulumi.String("postgres-storage"),
										MountPath: pulumi.String("/var/lib/postgresql/data"),
									},
									&corev1.VolumeMountArgs{
										Name:      pulumi.String("init-scripts"),
										MountPath: pulumi.String("/docker-entrypoint-initdb.d"),
										ReadOnly:  pulumi.Bool(true),
									},
								},
								Resources: &corev1.ResourceRequirementsArgs{
									Requests: pulumi.StringMap{
										"cpu":    pulumi.String("200m"),
										"memory": pulumi.String("512Mi"),
									},
									Limits: pulumi.StringMap{
										"cpu":    pulumi.String("1000m"),
										"memory": pulumi.String("1Gi"),
									},
								},
								LivenessProbe: &corev1.ProbeArgs{
									Exec: &corev1.ExecActionArgs{
										Command: pulumi.StringArray{
											pulumi.String("pg_isready"),
											pulumi.String("-U"),
											pulumi.String("kellogg_user"),
											pulumi.String("-d"),
											pulumi.String("kellogg_music_match"),
										},
									},
									InitialDelaySeconds: pulumi.Int(30),
									PeriodSeconds:       pulumi.Int(10),
									TimeoutSeconds:      pulumi.Int(5),
									FailureThreshold:    pulumi.Int(3),
								},
								ReadinessProbe: &corev1.ProbeArgs{
									Exec: &corev1.ExecActionArgs{
										Command: pulumi.StringArray{
											pulumi.String("pg_isready"),
											pulumi.String("-U"),
											pulumi.String("kellogg_user"),
											pulumi.String("-d"),
											pulumi.String("kellogg_music_match"),
										},
									},
									InitialDelaySeconds: pulumi.Int(5),
									PeriodSeconds:       pulumi.Int(5),
									TimeoutSeconds:      pulumi.Int(3),
									FailureThreshold:    pulumi.Int(3),
								},
							},
						},
						Volumes: corev1.VolumeArray{
							&corev1.VolumeArgs{
								Name: pulumi.String("init-scripts"),
								ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
									Name: pgConfigMap.Metadata.Name(),
									Items: corev1.KeyToPathArray{
										&corev1.KeyToPathArgs{
											Key:  pulumi.String("init-database.sh"),
											Path: pulumi.String("init-database.sh"),
											Mode: pulumi.Int(0755),
										},
										&corev1.KeyToPathArgs{
											Key:  pulumi.String("DATABASE_SCHEMA.sql"),
											Path: pulumi.String("DATABASE_SCHEMA.sql"),
											Mode: pulumi.Int(0644),
										},
									},
								},
							},
						},
					},
				},
				VolumeClaimTemplates: corev1.PersistentVolumeClaimTypeArray{
					&corev1.PersistentVolumeClaimTypeArgs{
						Metadata: &metav1.ObjectMetaArgs{
							Name: pulumi.String("postgres-storage"),
						},
						Spec: &corev1.PersistentVolumeClaimSpecArgs{
							AccessModes: pulumi.StringArray{
								pulumi.String("ReadWriteOnce"),
							},
							Resources: &corev1.VolumeResourceRequirementsArgs{
								Requests: pulumi.StringMap{
									"storage": pulumi.String("10Gi"),
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

		// Create PostgreSQL Service
		pgService, err := corev1.NewService(ctx, "postgres-service", &corev1.ServiceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("postgres"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kellogg-music-match"),
					"component": pulumi.String("database"),
				},
			},
			Spec: &corev1.ServiceSpecArgs{
				Selector: pulumi.StringMap{
					"app":       pulumi.String("kellogg-music-match"),
					"component": pulumi.String("database"),
				},
				Ports: corev1.ServicePortArray{
					&corev1.ServicePortArgs{
						Name:       pulumi.String("postgres"),
						Port:       pulumi.Int(5432),
						TargetPort: pulumi.String("postgres"),
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
		ctx.Export("postgresStatefulSetName", pgStatefulSet.Metadata.Name())
		ctx.Export("postgresServiceName", pgService.Metadata.Name())
		ctx.Export("postgresSecretName", pgSecret.Metadata.Name())

		// Export application URLs
		ctx.Export("uiUrl", pulumi.String("http://kmm-ui.traefik.me"))
		ctx.Export("backendUrl", pulumi.String("http://kmm-backend.traefik.me"))

		// Export ingress status (will show external IP when available)
		ctx.Export("ingressStatus", ingress.Status)

		return nil
	})
}

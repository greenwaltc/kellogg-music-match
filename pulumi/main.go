package main

import (
	"strconv"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	netv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	pconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := pconfig.New(ctx, "")

		frontendImage := cfg.Get("frontendImage")
		if frontendImage == "" {
			frontendImage = "kmm-ui:latest"
		}
		backendImage := cfg.Get("backendImage")
		if backendImage == "" {
			backendImage = "kmm-backend:latest"
		}
		replicaStr := cfg.Get("replicas")
		replicas := 1
		if replicaStr != "" {
			if v, err := strconv.Atoi(replicaStr); err == nil && v > 0 {
				replicas = v
			}
		}
		frontendHost := cfg.Get("frontendHost")         // e.g. app.example.com
		backendHost := cfg.Get("backendHost")           // e.g. api.example.com
		tlsSecret := cfg.Get("tlsSecretName")           // optional existing TLS secret
		publicBackendURL := cfg.Get("backendPublicUrl") // optional full https URL to embed in config.json

		// --- Backend Deployment & Service ---
		backendLabels := pulumi.StringMap{"app": pulumi.String("backend"), "tier": pulumi.String("api")}
		_, err := appsv1.NewDeployment(ctx, "backend-deployment", &appsv1.DeploymentArgs{
			Metadata: &metav1.ObjectMetaArgs{Labels: backendLabels},
			Spec: appsv1.DeploymentSpecArgs{
				Replicas: pulumi.Int(replicas),
				Selector: &metav1.LabelSelectorArgs{MatchLabels: backendLabels},
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{Labels: backendLabels},
					Spec: &corev1.PodSpecArgs{Containers: corev1.ContainerArray{
						corev1.ContainerArgs{
							Name:           pulumi.String("backend"),
							Image:          pulumi.String(backendImage),
							Ports:          corev1.ContainerPortArray{corev1.ContainerPortArgs{ContainerPort: pulumi.Int(8080)}},
							LivenessProbe:  &corev1.ProbeArgs{HttpGet: &corev1.HTTPGetActionArgs{Path: pulumi.String("/health"), Port: pulumi.Int(8080)}, InitialDelaySeconds: pulumi.Int(5), PeriodSeconds: pulumi.Int(15)},
							ReadinessProbe: &corev1.ProbeArgs{HttpGet: &corev1.HTTPGetActionArgs{Path: pulumi.String("/health"), Port: pulumi.Int(8080)}, InitialDelaySeconds: pulumi.Int(3), PeriodSeconds: pulumi.Int(10)},
						},
					}},
				},
			},
		})
		if err != nil {
			return err
		}

		backendSvc, err := corev1.NewService(ctx, "backend-service", &corev1.ServiceArgs{
			Metadata: &metav1.ObjectMetaArgs{Labels: backendLabels},
			Spec: &corev1.ServiceSpecArgs{
				Selector: backendLabels,
				Ports:    corev1.ServicePortArray{corev1.ServicePortArgs{Port: pulumi.Int(8080), TargetPort: pulumi.Int(8080)}},
				Type:     pulumi.StringPtr("ClusterIP"),
			},
		})
		if err != nil {
			return err
		}

		// --- Frontend ConfigMap (runtime config.json) ---
		// Frontend fetches /config.json; point API to in-cluster backend service name
		// Build config.json; prefer explicit public backend URL, then backendHost, else cluster service name.
		configJSON := pulumi.Sprintf("{\n  \"apiBaseUrl\": \"%s\"\n}", pulumi.All(backendSvc.Metadata.Name(), pulumi.String(publicBackendURL), pulumi.String(backendHost)).ApplyT(func(vals []interface{}) string {
			svcName := vals[0].(string)
			pub := vals[1].(string)
			bHost := vals[2].(string)
			switch {
			case pub != "":
				return pub
			case bHost != "":
				return "https://" + bHost
			default:
				return "http://" + svcName + ":8080"
			}
		}))

		frontCfg, err := corev1.NewConfigMap(ctx, "frontend-config", &corev1.ConfigMapArgs{
			Data: pulumi.StringMap{
				"config.json": configJSON,
			},
		})
		if err != nil {
			return err
		}

		// --- Frontend Deployment & Service ---
		frontendLabels := pulumi.StringMap{"app": pulumi.String("frontend"), "tier": pulumi.String("web")}
		_, err = appsv1.NewDeployment(ctx, "frontend-deployment", &appsv1.DeploymentArgs{
			Metadata: &metav1.ObjectMetaArgs{Labels: frontendLabels},
			Spec: appsv1.DeploymentSpecArgs{
				Replicas: pulumi.Int(replicas),
				Selector: &metav1.LabelSelectorArgs{MatchLabels: frontendLabels},
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{Labels: frontendLabels},
					Spec: &corev1.PodSpecArgs{Containers: corev1.ContainerArray{
						corev1.ContainerArgs{
							Name:           pulumi.String("frontend"),
							Image:          pulumi.String(frontendImage),
							Ports:          corev1.ContainerPortArray{corev1.ContainerPortArgs{ContainerPort: pulumi.Int(80)}},
							VolumeMounts:   corev1.VolumeMountArray{corev1.VolumeMountArgs{Name: pulumi.String("config"), MountPath: pulumi.String("/usr/share/nginx/html/config.json"), SubPath: pulumi.String("config.json"), ReadOnly: pulumi.Bool(true)}},
							LivenessProbe:  &corev1.ProbeArgs{HttpGet: &corev1.HTTPGetActionArgs{Path: pulumi.String("/health"), Port: pulumi.Int(80)}, InitialDelaySeconds: pulumi.Int(5), PeriodSeconds: pulumi.Int(30)},
							ReadinessProbe: &corev1.ProbeArgs{HttpGet: &corev1.HTTPGetActionArgs{Path: pulumi.String("/health"), Port: pulumi.Int(80)}, InitialDelaySeconds: pulumi.Int(3), PeriodSeconds: pulumi.Int(15)},
						},
					}, Volumes: corev1.VolumeArray{corev1.VolumeArgs{Name: pulumi.String("config"), ConfigMap: &corev1.ConfigMapVolumeSourceArgs{Name: frontCfg.Metadata.Name()}}}},
				},
			},
		})
		if err != nil {
			return err
		}

		frontSvc, err := corev1.NewService(ctx, "frontend-service", &corev1.ServiceArgs{
			Metadata: &metav1.ObjectMetaArgs{Labels: frontendLabels},
			Spec: &corev1.ServiceSpecArgs{
				Selector: frontendLabels,
				Ports:    corev1.ServicePortArray{corev1.ServicePortArgs{Port: pulumi.Int(80), TargetPort: pulumi.Int(80)}},
				// Use LoadBalancer for cloud; override via stack config if needed later
				Type: pulumi.StringPtr("LoadBalancer"),
			},
		})
		if err != nil {
			return err
		}

		// --- Ingress (optional) ---
		// If frontendHost is provided, create ingress rule for UI.
		var ingFrontend *netv1.Ingress
		if frontendHost != "" {
			ingFrontend, err = netv1.NewIngress(ctx, "frontend-ingress", &netv1.IngressArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Annotations: pulumi.StringMap{
						// Adjust class annotation to match your ingress controller if needed
						"kubernetes.io/ingress.class": pulumi.String("nginx"),
					},
				},
				Spec: &netv1.IngressSpecArgs{
					Tls: buildTLS(tlsSecret, frontendHost),
					Rules: netv1.IngressRuleArray{
						netv1.IngressRuleArgs{
							Host: pulumi.String(frontendHost),
							Http: &netv1.HTTPIngressRuleValueArgs{
								Paths: netv1.HTTPIngressPathArray{
									netv1.HTTPIngressPathArgs{
										Path:     pulumi.String("/"),
										PathType: pulumi.String("Prefix"),
										Backend: netv1.IngressBackendArgs{
											Service: &netv1.IngressServiceBackendArgs{
												Name: frontSvc.Metadata.Name().Elem(),
												Port: &netv1.ServiceBackendPortArgs{Number: pulumi.Int(80)},
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
			ctx.Export("frontendIngress", ingFrontend.Metadata.Name())
		}

		// Separate backend ingress (direct) if backendHost set
		var ingBackend *netv1.Ingress
		if backendHost != "" {
			ingBackend, err = netv1.NewIngress(ctx, "backend-ingress", &netv1.IngressArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Annotations: pulumi.StringMap{
						"kubernetes.io/ingress.class": pulumi.String("nginx"),
					},
				},
				Spec: &netv1.IngressSpecArgs{
					Tls: buildTLS(tlsSecret, backendHost),
					Rules: netv1.IngressRuleArray{
						netv1.IngressRuleArgs{
							Host: pulumi.String(backendHost),
							Http: &netv1.HTTPIngressRuleValueArgs{
								Paths: netv1.HTTPIngressPathArray{
									netv1.HTTPIngressPathArgs{
										Path:     pulumi.String("/"),
										PathType: pulumi.String("Prefix"),
										Backend: netv1.IngressBackendArgs{
											Service: &netv1.IngressServiceBackendArgs{
												Name: backendSvc.Metadata.Name().Elem(),
												Port: &netv1.ServiceBackendPortArgs{Number: pulumi.Int(8080)},
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
			ctx.Export("backendIngress", ingBackend.Metadata.Name())
		}

		// Exports
		ctx.Export("backendService", backendSvc.Metadata.Name())
		ctx.Export("frontendService", frontSvc.Metadata.Name())
		ctx.Export("frontendConfigMap", frontCfg.Metadata.Name())
		ctx.Export("images", pulumi.StringMap{"frontend": pulumi.String(frontendImage), "backend": pulumi.String(backendImage)})
		if frontendHost != "" {
			ctx.Export("frontendHost", pulumi.String(frontendHost))
		}
		if backendHost != "" {
			ctx.Export("backendHost", pulumi.String(backendHost))
		}
		if publicBackendURL != "" {
			ctx.Export("publicBackendUrl", pulumi.String(publicBackendURL))
		}
		return nil
	})
}

// ifTls returns a TLS configuration slice only if secret and host provided.
func buildTLS(secret, host string) netv1.IngressTLSArray {
	if secret == "" || host == "" {
		return netv1.IngressTLSArray{}
	}
	return netv1.IngressTLSArray{netv1.IngressTLSArgs{Hosts: pulumi.StringArray{pulumi.String(host)}, SecretName: pulumi.StringPtr(secret)}}
}

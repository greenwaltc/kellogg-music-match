package main

import (
	"os"
	"strings"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	networkingv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Read Flyway configuration file
		flywayConfig, err := os.ReadFile("../database/flyway.conf")
		if err != nil {
			return err
		}

		// Read migration files from the backend schema migrations directory
		migrationFiles := make(pulumi.StringMap)
		migrationDir := "../backend/db/schema/migrations"
		entries, err := os.ReadDir(migrationDir)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
				content, err := os.ReadFile(migrationDir + "/" + entry.Name())
				if err != nil {
					return err
				}
				migrationFiles[entry.Name()] = pulumi.String(string(content))
			}
		}
		// Create the namespace
		namespace, err := corev1.NewNamespace(ctx, "kmm", &corev1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("kmm"),
				Labels: pulumi.StringMap{
					"app": pulumi.String("kmm"),
				},
			},
		})
		if err != nil {
			return err
		}

		// Create the service account
		serviceAccount, err := corev1.NewServiceAccount(ctx, "kmm-sa", &corev1.ServiceAccountArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("kmm"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app": pulumi.String("kmm"),
				},
			},
		})
		if err != nil {
			return err
		}

		// Create ConfigMap for UI configuration with proxy-based backend URL
		uiConfigMap, err := corev1.NewConfigMap(ctx, "ui-config", &corev1.ConfigMapArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("kmm-ui-config"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kmm"),
					"component": pulumi.String("ui"),
				},
			},
			Data: pulumi.StringMap{
				"config.json": pulumi.String("{\"apiBaseUrl\": \"/api\", \"artistMinCount\": 5, \"artistMaxCount\": 20}"),
			},
		})
		if err != nil {
			return err
		}

		// Create Flyway ConfigMap with configuration and migration files (needed for backend init containers)
		flywayConfigMapData := pulumi.StringMap{
			"flyway.conf": pulumi.String(string(flywayConfig)),
		}

		// Add all migration files to the ConfigMap
		for filename, content := range migrationFiles {
			flywayConfigMapData[filename] = content
		}

		flywayConfigMap, err := corev1.NewConfigMap(ctx, "flyway-config", &corev1.ConfigMapArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("flyway-config"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kmm"),
					"component": pulumi.String("flyway"),
				},
			},
			Data: flywayConfigMapData,
		})
		if err != nil {
			return err
		}

		// Create ConfigMap for MusicBrainz data loading script only (CSV will be embedded in container)
		musicbrainzConfigMap, err := corev1.NewConfigMap(ctx, "musicbrainz-scripts", &corev1.ConfigMapArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("musicbrainz-scripts"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kmm"),
					"component": pulumi.String("data"),
				},
			},
			Data: pulumi.StringMap{
				"load_artists.sql": pulumi.String(`-- Load MusicBrainz artists data from embedded CSV
CREATE TEMP TABLE temp_musicbrainz_load (
    musicbrainz_id TEXT,
    name TEXT,
    sort_name TEXT,
    artist_type TEXT,
    gender TEXT,
    country TEXT,
    life_span_begin TEXT,
    life_span_end TEXT,
    disambiguation TEXT,
    musicbrainz_score TEXT
);

-- Load data from CSV file directly from embedded location using \copy (client-side)
\copy temp_musicbrainz_load FROM '/data/musicbrainz_artists_50k.csv' WITH (FORMAT csv, HEADER true, DELIMITER ',', QUOTE '"', ESCAPE '"');

-- Insert into artists table with proper type conversions and conflict handling
-- Only proceed if we have fewer than 1000 reference artists
DO $$
DECLARE
    existing_count INTEGER;
    loaded_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO existing_count FROM artists WHERE is_reference = TRUE;
    
    IF existing_count < 1000 THEN
        INSERT INTO artists (
            name, 
            musicbrainz_id, 
            sort_name, 
            artist_type, 
            gender, 
            country, 
            life_span_begin, 
            life_span_end, 
            disambiguation, 
            musicbrainz_score, 
            is_reference,
            created_at
        )
        SELECT DISTINCT ON (TRIM(name))
            TRIM(name),
            CASE WHEN TRIM(musicbrainz_id) = '' THEN NULL ELSE TRIM(musicbrainz_id)::UUID END,
            TRIM(sort_name),
            TRIM(artist_type),
            CASE WHEN TRIM(gender) = '' THEN NULL ELSE TRIM(gender) END,
            CASE WHEN TRIM(country) = '' THEN NULL ELSE TRIM(country) END,
            CASE WHEN TRIM(life_span_begin) = '' THEN NULL 
                 WHEN TRIM(life_span_begin) ~ '^\d{4}$' THEN (TRIM(life_span_begin) || '-01-01')::DATE
                 WHEN TRIM(life_span_begin) ~ '^\d{4}-\d{2}$' THEN (TRIM(life_span_begin) || '-01')::DATE
                 ELSE TRIM(life_span_begin)::DATE END,
            CASE WHEN TRIM(life_span_end) = '' THEN NULL 
                 WHEN TRIM(life_span_end) ~ '^\d{4}$' THEN (TRIM(life_span_end) || '-01-01')::DATE
                 WHEN TRIM(life_span_end) ~ '^\d{4}-\d{2}$' THEN (TRIM(life_span_end) || '-01')::DATE
                 ELSE TRIM(life_span_end)::DATE END,
            CASE WHEN TRIM(disambiguation) = '' THEN NULL ELSE TRIM(disambiguation) END,
            CASE WHEN TRIM(musicbrainz_score) = '' THEN NULL ELSE TRIM(musicbrainz_score)::INTEGER END,
            TRUE,
            CURRENT_TIMESTAMP
        FROM temp_musicbrainz_load
        WHERE TRIM(musicbrainz_id) != '' AND TRIM(name) != ''
        ORDER BY TRIM(name), musicbrainz_score DESC NULLS LAST
        ON CONFLICT (name) DO UPDATE SET
            musicbrainz_id = COALESCE(EXCLUDED.musicbrainz_id, artists.musicbrainz_id),
            sort_name = COALESCE(EXCLUDED.sort_name, artists.sort_name),
            artist_type = COALESCE(EXCLUDED.artist_type, artists.artist_type),
            gender = COALESCE(EXCLUDED.gender, artists.gender),
            country = COALESCE(EXCLUDED.country, artists.country),
            life_span_begin = COALESCE(EXCLUDED.life_span_begin, artists.life_span_begin),
            life_span_end = COALESCE(EXCLUDED.life_span_end, artists.life_span_end),
            disambiguation = COALESCE(EXCLUDED.disambiguation, artists.disambiguation),
            musicbrainz_score = COALESCE(EXCLUDED.musicbrainz_score, artists.musicbrainz_score),
            is_reference = TRUE;
        
        SELECT COUNT(*) INTO loaded_count FROM artists WHERE is_reference = TRUE;
        RAISE NOTICE 'Loaded % MusicBrainz reference artists (total: %)', 
            loaded_count - existing_count, loaded_count;
    ELSE
        RAISE NOTICE 'MusicBrainz data already exists (% reference artists), skipping load', existing_count;
    END IF;
END $$;

DROP TABLE temp_musicbrainz_load;`),
				"load_data.sh": pulumi.String(`#!/bin/bash
set -e

echo "🎵 Starting MusicBrainz data loading..."

# Wait for database to be ready
echo "Waiting for PostgreSQL to be ready..."
until pg_isready -h postgres -p 5432 -U kellogg_user; do
    echo "Waiting for postgres..."
    sleep 2
done

export PGPASSWORD=kellogg_secure_pass_2024

# Check if data already exists
echo "Checking if MusicBrainz data needs to be loaded..."
count=$(psql -h postgres -U kellogg_user -d kellogg_music_match -t -c "SELECT COUNT(*) FROM artists WHERE is_reference = TRUE;" | tr -d ' ')

if [ "$count" -lt 1000 ]; then
    echo "Loading MusicBrainz artists data..."
    echo "Found $count existing reference artists"
    
    # Verify the embedded CSV data exists and load it directly
    if [ -f "/data/musicbrainz_artists_50k.csv" ]; then
        echo "✅ Found CSV file at /data/musicbrainz_artists_50k.csv"
        echo "📊 File size: $(wc -l < /data/musicbrainz_artists_50k.csv) lines"
        psql -h postgres -U kellogg_user -d kellogg_music_match -f /scripts/load_artists.sql
        echo "✅ MusicBrainz data loaded successfully"
    else
        echo "❌ CSV file not found at /data/musicbrainz_artists_50k.csv"
        echo "📂 Available files in /data/:"
        ls -la /data/ || echo "No /data directory found"
        exit 1
    fi
else
    echo "✅ MusicBrainz data already exists ($count reference artists), skipping load"
fi

echo "🎉 MusicBrainz data loading completed"`),
			},
		})
		if err != nil {
			return err
		}

		// Create backend deployment with enhanced database integration
		backendDeployment, err := appsv1.NewDeployment(ctx, "backend-deployment", &appsv1.DeploymentArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("kmm-backend"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kmm"),
					"component": pulumi.String("backend"),
				},
			},
			Spec: &appsv1.DeploymentSpecArgs{
				Replicas: pulumi.Int(2),
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: pulumi.StringMap{
						"app":       pulumi.String("kmm"),
						"component": pulumi.String("backend"),
					},
				},
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Labels: pulumi.StringMap{
							"app":       pulumi.String("kmm"),
							"component": pulumi.String("backend"),
						},
					},
					Spec: &corev1.PodSpecArgs{
						ServiceAccountName: serviceAccount.Metadata.Name(),
						InitContainers: corev1.ContainerArray{
							&corev1.ContainerArgs{
								Name:  pulumi.String("wait-for-postgres"),
								Image: pulumi.String("postgres:16-alpine"),
								Command: pulumi.StringArray{
									pulumi.String("sh"),
									pulumi.String("-c"),
									pulumi.String("until pg_isready -h postgres -p 5432 -U kellogg_user; do echo waiting for postgres; sleep 2; done"),
								},
							},
							&corev1.ContainerArgs{
								Name:  pulumi.String("flyway-migrate"),
								Image: pulumi.String("flyway/flyway:latest"),
								Command: pulumi.StringArray{
									pulumi.String("flyway"),
									pulumi.String("migrate"),
								},
								Env: corev1.EnvVarArray{
									&corev1.EnvVarArgs{
										Name:  pulumi.String("FLYWAY_URL"),
										Value: pulumi.String("jdbc:postgresql://postgres:5432/kellogg_music_match"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("FLYWAY_USER"),
										Value: pulumi.String("kellogg_user"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("FLYWAY_PASSWORD"),
										Value: pulumi.String("kellogg_secure_pass_2024"),
									},
								},
								VolumeMounts: corev1.VolumeMountArray{
									&corev1.VolumeMountArgs{
										Name:      pulumi.String("flyway-config"),
										MountPath: pulumi.String("/flyway/conf"),
										ReadOnly:  pulumi.Bool(true),
									},
									&corev1.VolumeMountArgs{
										Name:      pulumi.String("flyway-migrations"),
										MountPath: pulumi.String("/flyway/sql"),
										ReadOnly:  pulumi.Bool(true),
									},
								},
							},
							&corev1.ContainerArgs{
								Name:            pulumi.String("load-musicbrainz-data"),
								Image:           pulumi.String("kellogg-music-match-musicbrainz:latest"),
								ImagePullPolicy: pulumi.String("Never"),
								Command: pulumi.StringArray{
									pulumi.String("bash"),
									pulumi.String("/scripts/load_data.sh"),
								},
								Env: corev1.EnvVarArray{
									&corev1.EnvVarArgs{
										Name:  pulumi.String("PGPASSWORD"),
										Value: pulumi.String("kellogg_secure_pass_2024"),
									},
								},
								VolumeMounts: corev1.VolumeMountArray{
									&corev1.VolumeMountArgs{
										Name:      pulumi.String("musicbrainz-scripts"),
										MountPath: pulumi.String("/scripts"),
										ReadOnly:  pulumi.Bool(true),
									},
								},
							},
						},
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
									// Server Configuration
									&corev1.EnvVarArgs{
										Name:  pulumi.String("SERVER_PORT"),
										Value: pulumi.String("8080"),
									},
									// Database Configuration
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
										Name:  pulumi.String("DB_SSLMODE"),
										Value: pulumi.String("disable"),
									},
									// CORS Configuration
									&corev1.EnvVarArgs{
										Name:  pulumi.String("CORS_ALLOWED_ORIGINS"),
										Value: pulumi.String("http://localhost:4200,http://kmm-ui.traefik.me,https://kmm-ui.traefik.me"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("CORS_ALLOWED_METHODS"),
										Value: pulumi.String("GET, POST, PUT, DELETE, OPTIONS"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("CORS_ALLOWED_HEADERS"),
										Value: pulumi.String("Content-Type, Authorization, X-User-Username"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("CORS_ALLOW_CREDENTIALS"),
										Value: pulumi.String("true"),
									},
									// Artist Configuration
									&corev1.EnvVarArgs{
										Name:  pulumi.String("ARTIST_MIN_COUNT"),
										Value: pulumi.String("5"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("ARTIST_MAX_COUNT"),
										Value: pulumi.String("20"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("ARTIST_MAX_NAME_LENGTH"),
										Value: pulumi.String("240"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("ARTIST_SEARCH_MAX_LENGTH"),
										Value: pulumi.String("240"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("ARTIST_SEARCH_LIMIT"),
										Value: pulumi.String("10"),
									},
									// Debug Configuration
									&corev1.EnvVarArgs{
										Name:  pulumi.String("DEBUG_ENABLED"),
										Value: pulumi.String("false"),
									},
									// Ticketmaster API Configuration
									&corev1.EnvVarArgs{
										Name:  pulumi.String("TICKETMASTER_CONSUMER_KEY"),
										Value: pulumi.String("3RVuRqbo6iLpQj0iEG6UUAZiWa2Z5Y0O"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("TICKETMASTER_CONSUMER_SECRET"),
										Value: pulumi.String("EzfZFlmQwTHXIrsb"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("TICKETMASTER_BASE_URL"),
										Value: pulumi.String("https://app.ticketmaster.com/discovery/v2"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("TICKETMASTER_TIMEOUT"),
										Value: pulumi.String("30"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("TICKETMASTER_MAX_RESULTS"),
										Value: pulumi.String("200"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("TICKETMASTER_DEFAULT_CITY"),
										Value: pulumi.String("Chicago"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("TICKETMASTER_DEFAULT_STATE"),
										Value: pulumi.String("IL"),
									},
									&corev1.EnvVarArgs{
										Name:  pulumi.String("TICKETMASTER_DEFAULT_COUNTRY"),
										Value: pulumi.String("US"),
									},
									// Legacy environment variables for backward compatibility
									&corev1.EnvVarArgs{
										Name:  pulumi.String("PORT"),
										Value: pulumi.String("8080"),
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
						Volumes: corev1.VolumeArray{
							&corev1.VolumeArgs{
								Name: pulumi.String("flyway-config"),
								ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
									Name: flywayConfigMap.Metadata.Name(),
								},
							},
							&corev1.VolumeArgs{
								Name: pulumi.String("flyway-migrations"),
								ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
									Name:        flywayConfigMap.Metadata.Name(),
									DefaultMode: pulumi.Int(0644),
								},
							},
							&corev1.VolumeArgs{
								Name: pulumi.String("musicbrainz-scripts"),
								ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
									Name:        musicbrainzConfigMap.Metadata.Name(),
									DefaultMode: pulumi.Int(0755),
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
				Name:      pulumi.String("kmm-backend"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kmm"),
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
					"app":       pulumi.String("kmm"),
					"component": pulumi.String("backend"),
				},
			},
		})
		if err != nil {
			return err
		}

		// Create UI deployment with Kellogg student profile support
		uiDeployment, err := appsv1.NewDeployment(ctx, "ui-deployment", &appsv1.DeploymentArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("kmm-ui"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kmm"),
					"component": pulumi.String("ui"),
				},
			},
			Spec: &appsv1.DeploymentSpecArgs{
				Replicas: pulumi.Int(2),
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: pulumi.StringMap{
						"app":       pulumi.String("kmm"),
						"component": pulumi.String("ui"),
					},
				},
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Labels: pulumi.StringMap{
							"app":       pulumi.String("kmm"),
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
				Name:      pulumi.String("kmm-ui"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kmm"),
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
					"app":       pulumi.String("kmm"),
					"component": pulumi.String("ui"),
				},
			},
		})
		if err != nil {
			return err
		}

		// Create ingress for both UI and backend
		ingress, err := networkingv1.NewIngress(ctx, "kmm-ingress", &networkingv1.IngressArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("kmm"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app": pulumi.String("kmm"),
				},
				Annotations: pulumi.StringMap{
					"traefik.ingress.kubernetes.io/router.entrypoints": pulumi.String("web,websecure"),
				},
			},
			Spec: &networkingv1.IngressSpecArgs{
				IngressClassName: pulumi.String("traefik"),
				Rules: networkingv1.IngressRuleArray{
					// UI ingress rule for traefik.me domain
					&networkingv1.IngressRuleArgs{
						Host: pulumi.String("kmm-ui.traefik.me"),
						Http: &networkingv1.HTTPIngressRuleValueArgs{
							Paths: networkingv1.HTTPIngressPathArray{
								&networkingv1.HTTPIngressPathArgs{
									Path:     pulumi.String("/"),
									PathType: pulumi.String("Prefix"),
									Backend: &networkingv1.IngressBackendArgs{
										Service: &networkingv1.IngressServiceBackendArgs{
											Name: pulumi.String("kmm-ui"),
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
											Name: pulumi.String("kmm-backend"),
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
					"app":       pulumi.String("kmm"),
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

		// Create PostgreSQL ConfigMap for environment variables only
		pgConfigMap, err := corev1.NewConfigMap(ctx, "postgres-config", &corev1.ConfigMapArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("postgres-config"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kmm"),
					"component": pulumi.String("database"),
				},
			},
			Data: pulumi.StringMap{
				"PGDATA": pulumi.String("/var/lib/postgresql/data/pgdata"),
			},
		})
		if err != nil {
			return err
		}

		// Create PostgreSQL StatefulSet with custom image (scientific extensions)
		pgStatefulSet, err := appsv1.NewStatefulSet(ctx, "postgres-statefulset", &appsv1.StatefulSetArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("postgres"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kmm"),
					"component": pulumi.String("database"),
				},
			},
			Spec: &appsv1.StatefulSetSpecArgs{
				ServiceName: pulumi.String("postgres"),
				Replicas:    pulumi.Int(1),
				Selector: &metav1.LabelSelectorArgs{
					MatchLabels: pulumi.StringMap{
						"app":       pulumi.String("kmm"),
						"component": pulumi.String("database"),
					},
				},
				Template: &corev1.PodTemplateSpecArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Labels: pulumi.StringMap{
							"app":       pulumi.String("kmm"),
							"component": pulumi.String("database"),
						},
					},
					Spec: &corev1.PodSpecArgs{
						ServiceAccountName: serviceAccount.Metadata.Name(),
						Containers: corev1.ContainerArray{
							&corev1.ContainerArgs{
								Name:            pulumi.String("postgres"),
								Image:           pulumi.String("kellogg-music-match-postgres:latest"),
								ImagePullPolicy: pulumi.String("Never"),
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
								},
								Resources: &corev1.ResourceRequirementsArgs{
									Requests: pulumi.StringMap{
										"cpu":    pulumi.String("200m"),
										"memory": pulumi.String("512Mi"),
									},
									Limits: pulumi.StringMap{
										"cpu":    pulumi.String("3000m"),
										"memory": pulumi.String("2Gi"),
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
								Name: pulumi.String("flyway-config"),
								ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
									Name: flywayConfigMap.Metadata.Name(),
									Items: corev1.KeyToPathArray{
										&corev1.KeyToPathArgs{
											Key:  pulumi.String("flyway.conf"),
											Path: pulumi.String("flyway.conf"),
											Mode: pulumi.Int(0644),
										},
									},
								},
							},
							&corev1.VolumeArgs{
								Name: pulumi.String("flyway-migrations"),
								ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
									Name:        flywayConfigMap.Metadata.Name(),
									DefaultMode: pulumi.Int(0644),
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
							StorageClassName: pulumi.String("local-path"),
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
					"app":       pulumi.String("kmm"),
					"component": pulumi.String("database"),
				},
			},
			Spec: &corev1.ServiceSpecArgs{
				Selector: pulumi.StringMap{
					"app":       pulumi.String("kmm"),
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

		// Export useful information for Kellogg Music Match deployment
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

		// Export application URLs for easy access
		ctx.Export("uiUrl", pulumi.String("http://kmm-ui.traefik.me"))
		ctx.Export("backendUrl", pulumi.String("http://kmm-backend.traefik.me"))
		ctx.Export("healthCheckUrl", pulumi.String("http://kmm-backend.traefik.me/health"))

		// Export database connection information
		ctx.Export("databaseHost", pulumi.String("postgres.kmm.svc.cluster.local"))
		ctx.Export("databasePort", pulumi.String("5432"))
		ctx.Export("databaseName", pulumi.String("kellogg_music_match"))
		ctx.Export("databaseUser", pulumi.String("kellogg_user"))

		// Export ingress status (will show external IP when available)
		ctx.Export("ingressStatus", ingress.Status)

		return nil
	})
}

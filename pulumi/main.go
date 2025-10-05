package main

import (
	"fmt"
	"os"
	"strings"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	networkingv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Helper accessors for Pulumi config (string-based for env vars)
		pulumiCfg := config.New(ctx, "")
		get := func(key, def string) pulumi.StringInput {
			if v := pulumiCfg.Get(key); v != "" {
				return pulumi.String(v)
			}
			return pulumi.String(def)
		}
		getInt := func(key string, def int) int {
			if v := pulumiCfg.GetInt(key); v != 0 {
				return v
			}
			return def
		}
		getBoolStr := func(key string, def bool) pulumi.StringInput {
			if v := pulumiCfg.GetBool(key); v {
				return pulumi.String("true")
			} else if pulumiCfg.Get(key) != "" {
				return pulumi.String("false")
			}
			if def {
				return pulumi.String("true")
			}
			return pulumi.String("false")
		}

		// Config-derived variables (defaults preserve existing hard-coded values)
		serverPort := get("serverPort", "8080")
		dbHost := get("dbHost", "postgres")
		dbPort := get("dbPort", "5432")
		dbUser := get("dbUser", "kellogg_user")
		// Sensitive database password must be provided as a Pulumi secret (no inline default)
		dbPassword := pulumiCfg.RequireSecret("dbPassword")
		dbName := get("dbName", "kellogg_music_match")
		dbSSLMode := get("dbSSLMode", "disable")
		corsAllowedOrigins := get("corsAllowedOrigins", "http://localhost:4200,http://kmm-ui.traefik.me,https://kmm-ui.traefik.me")
		corsAllowedMethods := get("corsAllowedMethods", "GET, POST, PUT, DELETE, OPTIONS")
		corsAllowedHeaders := get("corsAllowedHeaders", "Content-Type, Authorization, X-User-Username")
		corsAllowCredentials := getBoolStr("corsAllowCredentials", true)
		artistMinCount := get("artistMinCount", "5")
		artistMaxCount := get("artistMaxCount", "20")
		artistMaxNameLength := get("artistMaxNameLength", "240")
		artistSearchMaxLength := get("artistSearchMaxLength", "240")
		artistSearchLimit := get("artistSearchLimit", "10")
		debugEnabled := getBoolStr("debugEnabled", false)
		// Ticketmaster credentials required as secrets
		tmKey := pulumiCfg.RequireSecret("ticketmasterConsumerKey")
		tmSecret := pulumiCfg.RequireSecret("ticketmasterConsumerSecret")
		tmBaseURL := get("ticketmasterBaseUrl", "https://app.ticketmaster.com/discovery/v2")
		tmTimeout := get("ticketmasterTimeout", "30")
		tmMaxResults := get("ticketmasterMaxResults", "200")
		tmCity := get("ticketmasterDefaultCity", "Chicago")
		tmState := get("ticketmasterDefaultState", "IL")
		tmCountry := get("ticketmasterDefaultCountry", "US")
		emailEnabled := getBoolStr("emailEnabled", true)
		emailProvider := get("emailProvider", "sendgrid")
		emailFromEmail := get("emailFromEmail", "support@kelloggmatch.com")
		emailFromName := get("emailFromName", "Kellogg Music Match")
		appBaseURL := get("appBaseUrl", "https://kelloggmatch.com")
		// SendGrid API key secret
		sendgridAPIKey := pulumiCfg.RequireSecret("sendgridApiKey")
		smtpHost := get("smtpHost", "smtp.gmail.com")
		smtpPort := get("smtpPort", "587")
		smtpUser := get("smtpUser", "dummy-user@gmail.com")
		// SMTP password secret (if using SMTP path)
		smtpPass := pulumiCfg.RequireSecret("smtpPass")
		jwtSecret := pulumiCfg.RequireSecret("jwtSecretKey")
		jwtExpiry := get("jwtExpiryHours", "24")
		jwtRefresh := get("jwtRefreshHours", "720")
		legacyPort := serverPort
		// (DATABASE_URL no longer exported; application should build from discrete env vars)
		backendReplicas := getInt("backendReplicas", 2)
		uiReplicas := getInt("uiReplicas", 2)
		backendImage := get("backendImage", "kellogg-music-match-backend:latest")
		uiImage := get("uiImage", "kellogg-music-match-ui:latest")
		postgresImage := get("postgresImage", "kellogg-music-match-postgres:latest")
		// Image pull policy (was hardcoded as "Never" in multiple places)
		imagePullPolicy := get("imagePullPolicy", "IfNotPresent")
		musicbrainzImage := get("musicbrainzImage", "kellogg-music-match-musicbrainz:latest")
		flywayImage := get("flywayImage", "flyway/flyway:latest")
		tracingEnabled := getBoolStr("tracingEnabled", true)
		tracingExporter := get("tracingExporter", "otlp")
		otlpEndpoint := get("otlpEndpoint", "http://otel-collector:4318")
		otelServiceName := get("otelServiceName", "kmm-backend")
		otelServiceVersion := get("otelServiceVersion", "1.0.0")
		otelResourceAttributes := get("otelResourceAttributes", "environment=dev,team=matching")
		otelTracesSampler := get("otelTracesSampler", "parentbased_traceidratio")
		otelTracesSamplerArg := get("otelTracesSamplerArg", "0.10")
		// collectorImage := get("otelCollectorImage", "otel/opentelemetry-collector:0.103.1")
		// collectorReplicas := getInt("otelCollectorReplicas", 1)
		// collectorConfigOverride := pulumiCfg.Get("otelCollectorConfig")
		promEnabled := pulumiCfg.GetBool("otelCollectorPrometheus")
		collectorPromPort := getInt("otelCollectorPrometheusPort", 8888)
		backendCPUReq := get("backendCpuRequest", "100m")
		backendMemReq := get("backendMemRequest", "128Mi")
		backendCPULimit := get("backendCpuLimit", "500m")
		backendMemLimit := get("backendMemLimit", "512Mi")
		// collectorCPUReq := get("otelCollectorCpuRequest", "50m")
		// collectorMemReq := get("otelCollectorMemRequest", "64Mi")
		// collectorCPULimit := get("otelCollectorCpuLimit", "500m")
		// collectorMemLimit := get("otelCollectorMemLimit", "256Mi")
		postgresCPUReq := get("postgresCpuRequest", "200m")
		postgresMemReq := get("postgresMemRequest", "512Mi")
		postgresCPULimit := get("postgresCpuLimit", "3000m")
		postgresMemLimit := get("postgresMemLimit", "2Gi")
		uiCPUReq := get("uiCpuRequest", "50m")
		uiMemReq := get("uiMemRequest", "64Mi")
		uiCPULimit := get("uiCpuLimit", "200m")
		uiMemLimit := get("uiMemLimit", "256Mi")

		baseCollector := "receivers:\n  otlp:\n    protocols:\n      http:\n        endpoint: 0.0.0.0:4318\n      grpc:\nprocessors:\n  batch: {}\nexporters:\n  logging: {}\n"
		if promEnabled {
			baseCollector += fmt.Sprintf("  prometheus:\n    endpoint: 0.0.0.0:%d\n", collectorPromPort)
		}
		// Build traces pipeline exporters list (exclude prometheus exporter which doesn't handle traces)
		// baseCollector += "service:\n  pipelines:\n    traces:\n      receivers: [otlp]\n      processors: [batch]\n      exporters: [logging]\n"
		// collectorConfigContent := baseCollector
		// if collectorConfigOverride != "" {
		// 	collectorConfigContent = collectorConfigOverride
		// }

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
set -euo pipefail

echo "🎵 Starting MusicBrainz data loading..."

# Default/fallback values if not provided
DB_HOST="${DB_HOST:-postgres}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-kellogg_user}"
DB_NAME="${DB_NAME:-kellogg_music_match}"

echo "Using database $DB_USER@$DB_HOST:$DB_PORT/$DB_NAME"

# Wait for database to be ready
echo "Waiting for PostgreSQL to be ready..."
until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER"; do
	echo "Waiting for postgres..."
	sleep 2
done

# Prefer explicit PGPASSWORD but fall back to DB_PASSWORD from secret
export PGPASSWORD="${PGPASSWORD:-${DB_PASSWORD:-}}"

if [ -z "$PGPASSWORD" ]; then
  echo "❌ No database password provided (PGPASSWORD/DB_PASSWORD). Exiting." >&2
  exit 1
fi

# Check if data already exists
echo "Checking if MusicBrainz data needs to be loaded..."
count=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM artists WHERE is_reference = TRUE;" | tr -d ' ')
count=${count:-0}

if [ "$count" -lt 1000 ]; then
	echo "Loading MusicBrainz artists data..."
	echo "Found $count existing reference artists"
    
	# Verify the embedded CSV data exists and load it directly
	if [ -f "/data/musicbrainz_artists_50k.csv" ]; then
		echo "✅ Found CSV file at /data/musicbrainz_artists_50k.csv"
		echo "📊 File size: $(wc -l < /data/musicbrainz_artists_50k.csv) lines"
		psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f /scripts/load_artists.sql
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
		// First, create a Secret for backend-sensitive configuration values.
		backendSecret, err := corev1.NewSecret(ctx, "backend-secret", &corev1.SecretArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("backend-secret"),
				Namespace: namespace.Metadata.Name(),
				Labels: pulumi.StringMap{
					"app":       pulumi.String("kmm"),
					"component": pulumi.String("backend"),
				},
			},
			StringData: pulumi.StringMap{
				"JWT_SECRET_KEY":               jwtSecret.ToStringOutput(),
				"TICKETMASTER_CONSUMER_KEY":    tmKey.ToStringOutput(),
				"TICKETMASTER_CONSUMER_SECRET": tmSecret.ToStringOutput(),
				"SENDGRID_API_KEY":             sendgridAPIKey.ToStringOutput(),
				"SMTP_USER":                    smtpUser, // not secret
				"SMTP_PASS":                    smtpPass.ToStringOutput(),
				"DB_PASSWORD":                  dbPassword.ToStringOutput(),
				"FLYWAY_PASSWORD":              dbPassword.ToStringOutput(),
				"PGPASSWORD":                   dbPassword.ToStringOutput(),
			},
		})
		if err != nil {
			return err
		}

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
				Replicas: pulumi.Int(backendReplicas),
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
								Image: flywayImage,
								Command: pulumi.StringArray{
									pulumi.String("flyway"),
									pulumi.String("migrate"),
								},
								Env: corev1.EnvVarArray{
									&corev1.EnvVarArgs{Name: pulumi.String("FLYWAY_URL"), Value: pulumi.String("jdbc:postgresql://postgres:5432/kellogg_music_match")},
									&corev1.EnvVarArgs{Name: pulumi.String("FLYWAY_USER"), Value: pulumi.String("kellogg_user")},
									// FLYWAY_PASSWORD comes from secret via EnvFrom
								},
								EnvFrom: corev1.EnvFromSourceArray{
									&corev1.EnvFromSourceArgs{SecretRef: &corev1.SecretEnvSourceArgs{Name: backendSecret.Metadata.Name()}},
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
								Image:           musicbrainzImage,
								ImagePullPolicy: imagePullPolicy,
								Command: pulumi.StringArray{
									pulumi.String("bash"),
									pulumi.String("/scripts/load_data.sh"),
								},
								Env: corev1.EnvVarArray{
									&corev1.EnvVarArgs{Name: pulumi.String("DB_HOST"), Value: dbHost},
									&corev1.EnvVarArgs{Name: pulumi.String("DB_PORT"), Value: dbPort},
									&corev1.EnvVarArgs{Name: pulumi.String("DB_USER"), Value: dbUser},
									&corev1.EnvVarArgs{Name: pulumi.String("DB_NAME"), Value: dbName},
								},
								// PGPASSWORD comes from secret via EnvFrom
								EnvFrom: corev1.EnvFromSourceArray{
									&corev1.EnvFromSourceArgs{SecretRef: &corev1.SecretEnvSourceArgs{Name: backendSecret.Metadata.Name()}},
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
								Image:           backendImage,
								ImagePullPolicy: imagePullPolicy,
								Ports: corev1.ContainerPortArray{
									&corev1.ContainerPortArgs{
										ContainerPort: pulumi.Int(8080),
										Name:          pulumi.String("http"),
									},
								},
								// Pull all secret values in (DB_PASSWORD, JWT_SECRET_KEY, etc.)
								EnvFrom: corev1.EnvFromSourceArray{
									&corev1.EnvFromSourceArgs{SecretRef: &corev1.SecretEnvSourceArgs{Name: backendSecret.Metadata.Name()}},
								},
								Env: corev1.EnvVarArray{
									// Server Configuration
									&corev1.EnvVarArgs{Name: pulumi.String("SERVER_PORT"), Value: serverPort},
									// Database Configuration
									&corev1.EnvVarArgs{Name: pulumi.String("DB_HOST"), Value: dbHost},
									&corev1.EnvVarArgs{Name: pulumi.String("DB_PORT"), Value: dbPort},
									&corev1.EnvVarArgs{Name: pulumi.String("DB_USER"), Value: dbUser},
									// DB_PASSWORD provided via EnvFrom secret
									&corev1.EnvVarArgs{Name: pulumi.String("DB_NAME"), Value: dbName},
									&corev1.EnvVarArgs{Name: pulumi.String("DB_SSLMODE"), Value: dbSSLMode},
									// CORS Configuration
									&corev1.EnvVarArgs{Name: pulumi.String("CORS_ALLOWED_ORIGINS"), Value: corsAllowedOrigins},
									&corev1.EnvVarArgs{Name: pulumi.String("CORS_ALLOWED_METHODS"), Value: corsAllowedMethods},
									&corev1.EnvVarArgs{Name: pulumi.String("CORS_ALLOWED_HEADERS"), Value: corsAllowedHeaders},
									&corev1.EnvVarArgs{Name: pulumi.String("CORS_ALLOW_CREDENTIALS"), Value: corsAllowCredentials},
									// Artist Configuration
									&corev1.EnvVarArgs{Name: pulumi.String("ARTIST_MIN_COUNT"), Value: artistMinCount},
									&corev1.EnvVarArgs{Name: pulumi.String("ARTIST_MAX_COUNT"), Value: artistMaxCount},
									&corev1.EnvVarArgs{Name: pulumi.String("ARTIST_MAX_NAME_LENGTH"), Value: artistMaxNameLength},
									&corev1.EnvVarArgs{Name: pulumi.String("ARTIST_SEARCH_MAX_LENGTH"), Value: artistSearchMaxLength},
									&corev1.EnvVarArgs{Name: pulumi.String("ARTIST_SEARCH_LIMIT"), Value: artistSearchLimit},
									// Debug Configuration
									&corev1.EnvVarArgs{Name: pulumi.String("DEBUG_ENABLED"), Value: debugEnabled},
									// Ticketmaster API Configuration
									// Ticketmaster secrets provided via EnvFrom
									&corev1.EnvVarArgs{Name: pulumi.String("TICKETMASTER_BASE_URL"), Value: tmBaseURL},
									&corev1.EnvVarArgs{Name: pulumi.String("TICKETMASTER_TIMEOUT"), Value: tmTimeout},
									&corev1.EnvVarArgs{Name: pulumi.String("TICKETMASTER_MAX_RESULTS"), Value: tmMaxResults},
									&corev1.EnvVarArgs{Name: pulumi.String("TICKETMASTER_DEFAULT_CITY"), Value: tmCity},
									&corev1.EnvVarArgs{Name: pulumi.String("TICKETMASTER_DEFAULT_STATE"), Value: tmState},
									&corev1.EnvVarArgs{Name: pulumi.String("TICKETMASTER_DEFAULT_COUNTRY"), Value: tmCountry},
									// Email Configuration
									&corev1.EnvVarArgs{Name: pulumi.String("EMAIL_ENABLED"), Value: emailEnabled},
									&corev1.EnvVarArgs{Name: pulumi.String("EMAIL_PROVIDER"), Value: emailProvider},
									&corev1.EnvVarArgs{Name: pulumi.String("EMAIL_FROM_EMAIL"), Value: emailFromEmail},
									&corev1.EnvVarArgs{Name: pulumi.String("EMAIL_FROM_NAME"), Value: emailFromName},
									&corev1.EnvVarArgs{Name: pulumi.String("APP_BASE_URL"), Value: appBaseURL},
									// SendGrid Configuration (will be set when service is configured)
									// SENDGRID_API_KEY via EnvFrom
									// SMTP Configuration (alternative to SendGrid)
									&corev1.EnvVarArgs{Name: pulumi.String("SMTP_HOST"), Value: smtpHost},
									&corev1.EnvVarArgs{Name: pulumi.String("SMTP_PORT"), Value: smtpPort},
									// SMTP_USER / SMTP_PASS via EnvFrom
									// JWT Configuration
									// JWT_SECRET_KEY via EnvFrom
									&corev1.EnvVarArgs{Name: pulumi.String("JWT_EXPIRY_HOURS"), Value: jwtExpiry},
									&corev1.EnvVarArgs{Name: pulumi.String("JWT_REFRESH_HOURS"), Value: jwtRefresh},
									// Telemetry / Tracing Configuration
									&corev1.EnvVarArgs{Name: pulumi.String("TRACING_ENABLED"), Value: tracingEnabled},
									&corev1.EnvVarArgs{Name: pulumi.String("TRACING_EXPORTER"), Value: tracingExporter},
									&corev1.EnvVarArgs{Name: pulumi.String("OTEL_EXPORTER_OTLP_ENDPOINT"), Value: otlpEndpoint},
									&corev1.EnvVarArgs{Name: pulumi.String("OTEL_SERVICE_NAME"), Value: otelServiceName},
									&corev1.EnvVarArgs{Name: pulumi.String("OTEL_SERVICE_VERSION"), Value: otelServiceVersion},
									&corev1.EnvVarArgs{Name: pulumi.String("OTEL_RESOURCE_ATTRIBUTES"), Value: otelResourceAttributes},
									&corev1.EnvVarArgs{Name: pulumi.String("OTEL_TRACES_SAMPLER"), Value: otelTracesSampler},
									&corev1.EnvVarArgs{Name: pulumi.String("OTEL_TRACES_SAMPLER_ARG"), Value: otelTracesSamplerArg},
									// Legacy environment variables for backward compatibility
									&corev1.EnvVarArgs{Name: pulumi.String("PORT"), Value: legacyPort},
									// DATABASE_URL derived at runtime from other envs; omitted to avoid duplicating secret
								},
								Resources: &corev1.ResourceRequirementsArgs{
									Requests: pulumi.StringMap{"cpu": backendCPUReq, "memory": backendMemReq},
									Limits:   pulumi.StringMap{"cpu": backendCPULimit, "memory": backendMemLimit},
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
				Replicas: pulumi.Int(uiReplicas),
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
								Image:           uiImage,
								ImagePullPolicy: imagePullPolicy,
								Ports: corev1.ContainerPortArray{
									&corev1.ContainerPortArgs{
										ContainerPort: pulumi.Int(80),
										Name:          pulumi.String("http"),
									},
								},
								Resources: &corev1.ResourceRequirementsArgs{
									Requests: pulumi.StringMap{"cpu": uiCPUReq, "memory": uiMemReq},
									Limits:   pulumi.StringMap{"cpu": uiCPULimit, "memory": uiMemLimit},
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

		// Read ingress class from Pulumi config (default to traefik). This allows
		// stacks to override the controller (e.g. local development can set nginx).
		cfg := config.New(ctx, "")
		ingressClass := cfg.Get("ingressClass")
		if ingressClass == "" {
			ingressClass = "traefik"
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
					// Allow the ingress class to be configured per-stack. Default is
					// traefik; local stacks can set this to nginx.
					"kubernetes.io/ingress.class": pulumi.String(ingressClass),
				},
			},
			Spec: &networkingv1.IngressSpecArgs{
				// Use the configured IngressClass which exists in the cluster. This
				// defaults to "traefik" but can be overridden via Pulumi config
				// (e.g. set ingressClass=nginx for local development).
				IngressClassName: pulumi.String(ingressClass),
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
				"POSTGRES_USER":     dbUser,
				"POSTGRES_PASSWORD": dbPassword.ToStringOutput(),
				"POSTGRES_DB":       dbName,
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
								Image:           postgresImage,
								ImagePullPolicy: imagePullPolicy,
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
									Requests: pulumi.StringMap{"cpu": postgresCPUReq, "memory": postgresMemReq},
									Limits:   pulumi.StringMap{"cpu": postgresCPULimit, "memory": postgresMemLimit},
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

		// // OpenTelemetry Collector (ConfigMap + Deployment + Service) for OTLP ingestion
		// collectorConfigMap, err := corev1.NewConfigMap(ctx, "otel-collector-config", &corev1.ConfigMapArgs{
		// 	Metadata: &metav1.ObjectMetaArgs{
		// 		Name:      pulumi.String("otel-collector-config"),
		// 		Namespace: namespace.Metadata.Name(),
		// 		Labels:    pulumi.StringMap{"app": pulumi.String("kmm"), "component": pulumi.String("otel-collector")},
		// 	},
		// 	Data: pulumi.StringMap{"collector.yaml": pulumi.String(collectorConfigContent)},
		// })
		// if err != nil {
		// 	return err
		// }

		// collectorDeployment, err := appsv1.NewDeployment(ctx, "otel-collector", &appsv1.DeploymentArgs{
		// 	Metadata: &metav1.ObjectMetaArgs{
		// 		Name:      pulumi.String("otel-collector"),
		// 		Namespace: namespace.Metadata.Name(),
		// 		Labels:    pulumi.StringMap{"app": pulumi.String("kmm"), "component": pulumi.String("otel-collector")},
		// 	},
		// 	Spec: &appsv1.DeploymentSpecArgs{
		// 		Replicas: pulumi.Int(collectorReplicas),
		// 		Selector: &metav1.LabelSelectorArgs{MatchLabels: pulumi.StringMap{"app": pulumi.String("kmm"), "component": pulumi.String("otel-collector")}},
		// 		Template: &corev1.PodTemplateSpecArgs{
		// 			Metadata: &metav1.ObjectMetaArgs{Labels: pulumi.StringMap{"app": pulumi.String("kmm"), "component": pulumi.String("otel-collector")}},
		// 			Spec: &corev1.PodSpecArgs{
		// 				ServiceAccountName: serviceAccount.Metadata.Name(),
		// 				Containers: corev1.ContainerArray{
		// 					&corev1.ContainerArgs{
		// 						Name:  pulumi.String("otel-collector"),
		// 						Image: collectorImage,
		// 						Args:  pulumi.StringArray{pulumi.String("--config=/conf/collector.yaml")},
		// 						Ports: corev1.ContainerPortArray{
		// 							&corev1.ContainerPortArgs{ContainerPort: pulumi.Int(4318), Name: pulumi.String("otlp-http")},
		// 							&corev1.ContainerPortArgs{ContainerPort: pulumi.Int(4317), Name: pulumi.String("otlp-grpc")},
		// 						},
		// 						VolumeMounts: corev1.VolumeMountArray{
		// 							&corev1.VolumeMountArgs{Name: pulumi.String("otel-config"), MountPath: pulumi.String("/conf")},
		// 						},
		// 						Resources:      &corev1.ResourceRequirementsArgs{Requests: pulumi.StringMap{"cpu": collectorCPUReq, "memory": collectorMemReq}, Limits: pulumi.StringMap{"cpu": collectorCPULimit, "memory": collectorMemLimit}},
		// 						LivenessProbe:  &corev1.ProbeArgs{HttpGet: &corev1.HTTPGetActionArgs{Path: pulumi.String("/healthz"), Port: pulumi.String("otlp-http")}, InitialDelaySeconds: pulumi.Int(15), PeriodSeconds: pulumi.Int(30)},
		// 						ReadinessProbe: &corev1.ProbeArgs{HttpGet: &corev1.HTTPGetActionArgs{Path: pulumi.String("/healthz"), Port: pulumi.String("otlp-http")}, InitialDelaySeconds: pulumi.Int(5), PeriodSeconds: pulumi.Int(15)},
		// 					},
		// 				},
		// 				Volumes: corev1.VolumeArray{
		// 					&corev1.VolumeArgs{Name: pulumi.String("otel-config"), ConfigMap: &corev1.ConfigMapVolumeSourceArgs{Name: collectorConfigMap.Metadata.Name()}},
		// 				},
		// 			},
		// 		},
		// 	},
		// })
		// if err != nil {
		// 	return err
		// }

		// collectorService, err := corev1.NewService(ctx, "otel-collector-svc", &corev1.ServiceArgs{
		// 	Metadata: &metav1.ObjectMetaArgs{Name: pulumi.String("otel-collector"), Namespace: namespace.Metadata.Name(), Labels: pulumi.StringMap{"app": pulumi.String("kmm"), "component": pulumi.String("otel-collector")}},
		// 	Spec: &corev1.ServiceSpecArgs{
		// 		Type: pulumi.String("ClusterIP"),
		// 		Ports: corev1.ServicePortArray{
		// 			&corev1.ServicePortArgs{Name: pulumi.String("otlp-http"), Port: pulumi.Int(4318), TargetPort: pulumi.String("otlp-http")},
		// 			&corev1.ServicePortArgs{Name: pulumi.String("otlp-grpc"), Port: pulumi.Int(4317), TargetPort: pulumi.String("otlp-grpc")},
		// 		},
		// 		Selector: pulumi.StringMap{"app": pulumi.String("kmm"), "component": pulumi.String("otel-collector")},
		// 	},
		// })
		// if err != nil {
		// 	return err
		// }

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
		// ctx.Export("otelCollectorDeployment", collectorDeployment.Metadata.Name())
		// ctx.Export("otelCollectorService", collectorService.Metadata.Name())

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

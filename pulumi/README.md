# Kellogg Music Match - Kubernetes Deployment

This Pulumi program deploys the Kellogg Music Match application to a Kubernetes cluster with consolidated database schema, enhanced Kellogg student features, and scientific similarity calculations.

## Prerequisites

1. **Pulumi CLI**: Install from [pulumi.com](https://www.pulumi.com/docs/get-started/install/)
2. **kubectl**: Configured to connect to your Kubernetes cluster
3. **Docker Images**: Ensure your application images are built and available:
   - `kellogg-music-match-postgres:latest` (Custom PostgreSQL 15 with scientific extensions)
   - `kellogg-music-match-backend:latest` (Go backend with enhanced features)
   - `kellogg-music-match-ui:latest` (Angular UI with Kellogg student profiles)
4. **Ingress Controller**: NGINX Ingress Controller installed in your cluster

## Building Docker Images

Before deploying, build and tag your Docker images:

```bash
# From the project root
cd ../

# Build custom PostgreSQL image with scientific libraries and consolidated schema
docker build -f postgres.dockerfile -t kellogg-music-match-postgres:latest .

# Build backend image with enhanced database features
docker build -t kellogg-music-match-backend:latest ./backend

# Build UI image with Kellogg student profile support
docker build -t kellogg-music-match-ui:latest ./ui

# If using a container registry, tag and push:
# docker tag kellogg-music-match-postgres:latest your-registry/kellogg-music-match-postgres:latest
# docker tag kellogg-music-match-backend:latest your-registry/kellogg-music-match-backend:latest
# docker tag kellogg-music-match-ui:latest your-registry/kellogg-music-match-ui:latest
# docker push your-registry/kellogg-music-match-postgres:latest
# docker push your-registry/kellogg-music-match-backend:latest
# docker push your-registry/kellogg-music-match-ui:latest
```

## Deployment

1. **Initialize Pulumi stack**:
   ```bash
   pulumi stack init dev
   ```

2. **Configure Kubernetes context** (if needed):
   ```bash
   pulumi config set kubernetes:context your-k8s-context
   ```

3. **Deploy the application**:
   ```bash
   pulumi up
   ```

4. **View deployment status**:
   ```bash
   pulumi stack output
   ```

## Resources Created

The Pulumi program creates the following Kubernetes resources:

### Namespace
- **Name**: `kellogg-music-match`
- **Purpose**: Isolates all application resources

### Service Account
- **Name**: `kellogg-music-match`
- **Namespace**: `kellogg-music-match`
- **Purpose**: Provides identity for pods

### Backend Deployment & Service
- **Deployment**: `kellogg-music-match-backend`
- **Service**: `kellogg-music-match-backend`
- **Replicas**: 2
- **Port**: 8080
- **Health Checks**: `/health` endpoint
- **Features**: Enhanced SQLC integration, consolidated schema support
- **Database Environment Variables**: Pre-configured for PostgreSQL connection
- **Resources**: 
  - Requests: 100m CPU, 128Mi memory
  - Limits: 500m CPU, 512Mi memory

### PostgreSQL Database (Enhanced with Scientific Extensions)
- **StatefulSet**: `postgres`
- **Service**: `postgres`
- **Image**: `kellogg-music-match-postgres:latest` (Custom PostgreSQL 15 with scientific libraries)
- **Extensions**: plpython3u, scipy, numpy for advanced music similarity calculations
- **Schema**: Consolidated single migration file with Kellogg-specific enhancements
- **Port**: 5432
- **Storage**: 10Gi persistent volume
- **Database**: `kellogg_music_match`
- **User**: `kellogg_user`
- **Secret**: `postgres-secret` (contains credentials)
- **Health Checks**: `pg_isready` probes
- **Enhanced Features**: 
  - Kellogg student profiles with `program` and `graduation_year` fields
  - Program validation (2Y, 1Y, MMM, MBAi, JD-MBA, MD-MBA, EWMBA)
  - Graduation year constraints (2025-2030)
  - `spearman_distance()`: Hybrid Jaccard + positional correlation algorithm
  - Complete consolidated schema initialization
- **Resources**:
  - Requests: 200m CPU, 512Mi memory
  - Limits: 1000m CPU, 1Gi memory

### UI Deployment & Service
- **Deployment**: `kellogg-music-match-ui`
- **Service**: `kellogg-music-match-ui`
- **Replicas**: 2
- **Port**: 80
- **Health Checks**: `/` endpoint
- **Features**: Kellogg student registration with program and graduation year
- **Resources**:
  - Requests: 50m CPU, 64Mi memory
  - Limits: 200m CPU, 256Mi memory

### Ingress
- **Name**: `kellogg-music-match`
- **Class**: `nginx`
- **Routes**:
  - `/` → UI Service (port 80)
  - `/api` → Backend Service (port 8080)
  - `/health` → Backend Service (port 8080)

## Accessing the Application

After deployment, get the ingress IP:

```bash
kubectl get ingress -n kellogg-music-match
```

Or use Pulumi to check the status:

```bash
pulumi stack output ingressStatus
```

Access the application at:
- **UI**: `http://<ingress-ip>/`
- **Backend API**: `http://<ingress-ip>/api`
- **Health Check**: `http://<ingress-ip>/health`

## Configuration for Container Registry

If using a private container registry, update the image references in `main.go`:

```go
Image: pulumi.String("your-registry/kellogg-music-match-postgres:latest"),
Image: pulumi.String("your-registry/kellogg-music-match-backend:latest"),
Image: pulumi.String("your-registry/kellogg-music-match-ui:latest"),
```

**Important**: The custom PostgreSQL image with scientific libraries must be pushed to your registry:

```bash
# Build and push custom PostgreSQL image
docker build -f postgres.dockerfile -t your-registry/kellogg-music-match-postgres:latest .
docker push your-registry/kellogg-music-match-postgres:latest
```

And add image pull secrets if needed:

```go
ImagePullSecrets: corev1.LocalObjectReferenceArray{
    &corev1.LocalObjectReferenceArgs{
        Name: pulumi.String("your-registry-secret"),
    },
},
```

## Monitoring and Debugging

Check pod status:
```bash
kubectl get pods -n kellogg-music-match
```

View logs:
```bash
kubectl logs -n kellogg-music-match -l component=backend
kubectl logs -n kellogg-music-match -l component=ui
```

Check services:
```bash
kubectl get services -n kellogg-music-match
```

Check ingress:
```bash
kubectl describe ingress -n kellogg-music-match
```

## Database Access

### PostgreSQL Connection Information (Enhanced Scientific Database)
After deployment, use these connection details:

- **Host**: `postgres.kellogg-music-match.svc.cluster.local`
- **Port**: `5432`
- **Database**: `kellogg_music_match`
- **Username**: `kellogg_user`
- **Password**: Retrieved from `postgres-secret`
- **Schema**: Consolidated schema with Kellogg-specific enhancements
- **Extensions**: plpython3u, scipy, numpy available for advanced calculations
- **Enhanced Features**: Kellogg student profiles, program validation, scientific similarity functions

### Port Forward for Local Access
```bash
# Forward PostgreSQL port for administration
kubectl port-forward -n kellogg-music-match service/postgres 5432:5432

# Connect with psql
psql -h localhost -p 5432 -U kellogg_user -d kellogg_music_match
```

### Direct Pod Access
```bash
# Execute psql in the PostgreSQL pod
kubectl exec -it -n kellogg-music-match postgres-0 -- psql -U kellogg_user -d kellogg_music_match

# Test scientific similarity function
kubectl exec -it -n kellogg-music-match postgres-0 -- psql -U kellogg_user -d kellogg_music_match -c "SELECT spearman_distance(ARRAY['Tool', 'Radiohead'], ARRAY['Tool', 'Radiohead']);"

# Verify Kellogg student schema
kubectl exec -it -n kellogg-music-match postgres-0 -- psql -U kellogg_user -d kellogg_music_match -c "SELECT column_name FROM information_schema.columns WHERE table_name='users' AND column_name IN ('program', 'graduation_year');"

# Verify Python scientific libraries
kubectl exec -it -n kellogg-music-match postgres-0 -- python3 -c "import scipy.stats; import numpy; print('✅ Scientific libraries available')"
```

### Database Outputs
The Pulumi stack exports these database-related outputs:
- `databaseHost`: Internal cluster hostname
- `databasePort`: PostgreSQL port (5432)
- `databaseName`: Database name
- `databaseUser`: Database username
- `postgresStatefulSetName`: StatefulSet resource name
- `postgresServiceName`: Service resource name
- `postgresSecretName`: Secret containing credentials

## Cleanup

To remove all resources:

```bash
pulumi destroy
```

## Customization

### Scaling
Modify the `Replicas` field in the deployment specs to scale up/down.

### Resources
Adjust the `Resources` section for each container based on your cluster capacity and application needs.

### Ingress Annotations
Add additional annotations to the ingress for SSL termination, rate limiting, etc.:

```go
Annotations: pulumi.StringMap{
    "nginx.ingress.kubernetes.io/ssl-redirect":   pulumi.String("true"),
    "nginx.ingress.kubernetes.io/rate-limit":     pulumi.String("100"),
    "cert-manager.io/cluster-issuer":             pulumi.String("letsencrypt-prod"),
},
```
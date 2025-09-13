# Kellogg Music Match - Kubernetes Deployment

This Pulumi program deploys the Kellogg Music Match application to a Kubernetes cluster.

## Prerequisites

1. **Pulumi CLI**: Install from [pulumi.com](https://www.pulumi.com/docs/get-started/install/)
2. **kubectl**: Configured to connect to your Kubernetes cluster
3. **Docker Images**: Ensure your application images are built and available:
   - `kellogg-music-match-backend:latest`
   - `kellogg-music-match-ui:latest`
4. **Ingress Controller**: NGINX Ingress Controller installed in your cluster

## Building Docker Images

Before deploying, build and tag your Docker images:

```bash
# From the project root
cd ../

# Build backend image
docker build -t kellogg-music-match-backend:latest ./backend

# Build UI image
docker build -t kellogg-music-match-ui:latest ./ui

# If using a container registry, tag and push:
# docker tag kellogg-music-match-backend:latest your-registry/kellogg-music-match-backend:latest
# docker push your-registry/kellogg-music-match-backend:latest
# docker tag kellogg-music-match-ui:latest your-registry/kellogg-music-match-ui:latest
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
- **Resources**: 
  - Requests: 100m CPU, 128Mi memory
  - Limits: 500m CPU, 512Mi memory

### UI Deployment & Service
- **Deployment**: `kellogg-music-match-ui`
- **Service**: `kellogg-music-match-ui`
- **Replicas**: 2
- **Port**: 80
- **Health Checks**: `/` endpoint
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
Image: pulumi.String("your-registry/kellogg-music-match-backend:latest"),
Image: pulumi.String("your-registry/kellogg-music-match-ui:latest"),
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
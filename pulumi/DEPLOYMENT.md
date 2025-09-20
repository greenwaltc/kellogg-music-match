# Kellogg Music Match - Production Deployment Guide

This guide covers deploying the Kellogg Music Match application to production Kubernetes clusters with enhanced security, monitoring, and scalability considerations.

## 🚀 Production Deployment Checklist

### Prerequisites
- [ ] Kubernetes cluster with minimum 3 nodes
- [ ] Ingress controller (NGINX or Traefik) installed
- [ ] Certificate manager for SSL/TLS
- [ ] Container registry access
- [ ] Pulumi CLI installed and configured
- [ ] kubectl configured for target cluster

### Security Configuration
- [ ] Private container registry setup
- [ ] Image pull secrets configured
- [ ] Network policies implemented
- [ ] PostgreSQL passwords from external secret management
- [ ] RBAC policies defined
- [ ] SSL/TLS certificates configured

## 🔧 Production Configuration

### 1. Container Registry Setup

Update image references in `main.go` for your registry:

```go
// Update these image references
Image: pulumi.String("your-registry.com/kellogg-music-match/postgres:latest"),
Image: pulumi.String("your-registry.com/kellogg-music-match/backend:latest"),
Image: pulumi.String("your-registry.com/kellogg-music-match/ui:latest"),
```

### 2. Environment-Specific Configuration

Create production stack configuration:

```bash
# Create production stack
pulumi stack init production

# Configure production-specific settings
pulumi config set kubernetes:namespace kellogg-music-match-prod
pulumi config set replicas:backend 3
pulumi config set replicas:ui 2
pulumi config set storage:size 50Gi
pulumi config set ingress:domain music-match.kellogg.northwestern.edu
```

### 3. Database Security

For production, use external secret management:

```go
// Example: Use external secret instead of inline values
pgSecret, err := corev1.NewSecret(ctx, "postgres-secret", &corev1.SecretArgs{
    // Reference external secret management system
    StringData: pulumi.StringMap{
        "POSTGRES_USER":     config.Require("database:user"),
        "POSTGRES_PASSWORD": config.RequireSecret("database:password"),
        "POSTGRES_DB":       config.Require("database:name"),
    },
})
```

### 4. SSL/TLS Configuration

Update ingress with TLS:

```go
ingress, err := networkingv1.NewIngress(ctx, "kellogg-music-match-ingress", &networkingv1.IngressArgs{
    Metadata: &metav1.ObjectMetaArgs{
        Annotations: pulumi.StringMap{
            "cert-manager.io/cluster-issuer": pulumi.String("letsencrypt-prod"),
            "nginx.ingress.kubernetes.io/ssl-redirect": pulumi.String("true"),
        },
    },
    Spec: &networkingv1.IngressSpecArgs{
        Tls: networkingv1.IngressTLSArray{
            &networkingv1.IngressTLSArgs{
                Hosts: pulumi.StringArray{
                    pulumi.String("music-match.kellogg.northwestern.edu"),
                },
                SecretName: pulumi.String("kellogg-music-match-tls"),
            },
        },
        // ... rest of spec
    },
})
```

## 📊 Monitoring and Observability

### 1. Health Checks
Enhanced health checks are already configured:
- **Backend**: `/health` endpoint with database connectivity check
- **Database**: `pg_isready` probes
- **UI**: Basic HTTP availability check

### 2. Resource Monitoring
Monitor these key metrics:
- PostgreSQL connection pool usage
- Backend response times for `/findMusicMatches` endpoint
- UI loading performance
- Database query performance for similarity calculations

### 3. Logging
Application logs are available via:
```bash
# Backend logs (includes similarity calculation timing)
kubectl logs -n kellogg-music-match -l component=backend

# Database logs (scientific function usage)
kubectl logs -n kellogg-music-match -l component=database

# UI logs
kubectl logs -n kellogg-music-match -l component=ui
```

## 🔄 Deployment Process

### 1. Pre-deployment Validation
```bash
# Build and test images locally
make docker-build
make test

# Validate Pulumi configuration
pulumi preview --stack production
```

### 2. Deploy to Production
```bash
# Deploy with confirmation
pulumi up --stack production --yes

# Verify deployment
kubectl get pods -n kellogg-music-match-prod
kubectl get services -n kellogg-music-match-prod
kubectl get ingress -n kellogg-music-match-prod
```

### 3. Post-deployment Verification
```bash
# Test health endpoints
curl https://music-match.kellogg.northwestern.edu/health

# Test user registration with Kellogg profile
curl -X POST https://music-match.kellogg.northwestern.edu/register \
  -H "Content-Type: application/json" \
  -d '{"username":"test","email":"test@kellogg.northwestern.edu","password":"Test123!","firstName":"Test","lastName":"User","program":"2Y","graduationYear":2026}'

# Test database scientific functions
kubectl exec -it -n kellogg-music-match-prod postgres-0 -- \
  psql -U kellogg_user -d kellogg_music_match \
  -c "SELECT spearman_distance(ARRAY['Tool', 'Radiohead'], ARRAY['Tool', 'Radiohead']);"
```

## 📈 Scaling Considerations

### 1. Backend Scaling
```go
// Increase backend replicas for high load
Replicas: pulumi.Int(5),

// Adjust resource limits
Resources: &corev1.ResourceRequirementsArgs{
    Requests: pulumi.StringMap{
        "cpu":    pulumi.String("500m"),
        "memory": pulumi.String("1Gi"),
    },
    Limits: pulumi.StringMap{
        "cpu":    pulumi.String("2000m"),
        "memory": pulumi.String("4Gi"),
    },
},
```

### 2. Database Scaling
For high-performance similarity calculations:
```go
// Increase PostgreSQL resources
Resources: &corev1.ResourceRequirementsArgs{
    Requests: pulumi.StringMap{
        "cpu":    pulumi.String("1000m"),
        "memory": pulumi.String("4Gi"),
    },
    Limits: pulumi.StringMap{
        "cpu":    pulumi.String("4000m"),
        "memory": pulumi.String("8Gi"),
    },
},
```

### 3. Storage Scaling
```go
// Increase persistent volume size
Resources: &corev1.VolumeResourceRequirementsArgs{
    Requests: pulumi.StringMap{
        "storage": pulumi.String("100Gi"), // Increased from 10Gi
    },
},
```

## 🔒 Security Best Practices

### 1. Network Policies
Implement network segmentation:
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: kellogg-music-match-network-policy
spec:
  podSelector:
    matchLabels:
      app: kellogg-music-match
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
```

### 2. Pod Security Standards
```go
// Add security context to containers
SecurityContext: &corev1.SecurityContextArgs{
    RunAsNonRoot: pulumi.Bool(true),
    RunAsUser:    pulumi.Int(1000),
    ReadOnlyRootFilesystem: pulumi.Bool(true),
},
```

### 3. Secret Management
Use external secret management (AWS Secrets Manager, Azure Key Vault, etc.):
```bash
# Example with External Secrets Operator
kubectl apply -f external-secret-postgres.yaml
```

## 🆘 Troubleshooting

### Common Issues

1. **PostgreSQL Scientific Functions Not Available**
   ```bash
   # Verify custom image is used
   kubectl describe pod -n kellogg-music-match postgres-0 | grep Image
   
   # Check if extensions are loaded
   kubectl exec -it postgres-0 -- psql -U kellogg_user -d kellogg_music_match -c "SELECT * FROM pg_extension;"
   ```

2. **Backend Database Connection Issues**
   ```bash
   # Check database connectivity
   kubectl exec -it deployment/kellogg-music-match-backend -- \
     nc -zv postgres 5432
   ```

3. **Ingress SSL/TLS Issues**
   ```bash
   # Check certificate status
   kubectl describe certificaterequest -n kellogg-music-match
   kubectl describe certificate -n kellogg-music-match
   ```

### Performance Optimization

1. **Database Query Optimization**
   - Monitor slow queries in PostgreSQL logs
   - Optimize indexes for music matching queries
   - Consider read replicas for high read loads

2. **Backend Performance**
   - Monitor `/findMusicMatches` endpoint response times
   - Consider caching for repeated similarity calculations
   - Scale horizontally with increased replicas

3. **UI Performance**
   - Use CDN for static assets
   - Implement client-side caching
   - Optimize bundle size for faster loading

## 📝 Maintenance

### Regular Tasks
- [ ] Update Docker images monthly
- [ ] Monitor certificate expiration
- [ ] Review resource usage and scale as needed
- [ ] Backup database regularly
- [ ] Test disaster recovery procedures
- [ ] Update dependencies and security patches

### Database Maintenance
```bash
# Regular database maintenance
kubectl exec -it postgres-0 -- psql -U kellogg_user -d kellogg_music_match -c "VACUUM ANALYZE;"

# Monitor database size and performance
kubectl exec -it postgres-0 -- psql -U kellogg_user -d kellogg_music_match -c "SELECT pg_size_pretty(pg_database_size('kellogg_music_match'));"
```
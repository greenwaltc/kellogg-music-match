# 🗄️ Database Setup - PostgreSQL

This document describes the PostgreSQL database setup for the Kellogg Music Match application, deployed as a Kubernetes StatefulSet for cloud-agnostic persistence.

## 📋 Database Overview

### Database Configuration
- **Database Name**: `kellogg_music_match`
- **Username**: `kellogg_user`
- **Password**: `secure_password_123` (stored in Kubernetes Secret)
- **Port**: `5432`
- **Host**: `postgres.kellogg-music-match.svc.cluster.local` (internal cluster DNS)

### Storage Configuration
- **Persistent Volume**: 10Gi storage allocated per StatefulSet replica
- **Access Mode**: ReadWriteOnce
- **Storage Class**: Uses cluster default storage class
- **Data Directory**: `/var/lib/postgresql/data/pgdata`

## 🏗️ Kubernetes Resources

### PostgreSQL StatefulSet
```yaml
Name: postgres
Namespace: kellogg-music-match
Replicas: 1
Image: postgres:15-alpine
```

**Key Features:**
- Persistent storage with volume claim templates
- Health checks with `pg_isready` probes
- Resource limits and requests configured
- Alpine-based image for minimal footprint

### Database Service
```yaml
Name: postgres
Type: ClusterIP
Port: 5432
```

**Access Pattern:**
- Internal cluster access only (ClusterIP)
- Backend connects via service DNS name
- No external exposure for security

### Secrets Management
```yaml
Secret: postgres-secret
Type: Opaque
```

**Stored Credentials:**
- `POSTGRES_DB`: Database name
- `POSTGRES_USER`: Database username  
- `POSTGRES_PASSWORD`: Database password

## 🔌 Backend Integration

The backend deployment is pre-configured with database environment variables for future migration:

### Environment Variables
```bash
DB_HOST=postgres.kellogg-music-match.svc.cluster.local
DB_PORT=5432
DB_NAME=kellogg_music_match        # From Secret
DB_USER=kellogg_user              # From Secret  
DB_PASSWORD=secure_password_123    # From Secret
DB_SSLMODE=disable                # For local development
```

### Connection String Example
```go
// Future Go database connection
dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
    os.Getenv("DB_HOST"),
    os.Getenv("DB_PORT"),
    os.Getenv("DB_USER"),
    os.Getenv("DB_PASSWORD"),
    os.Getenv("DB_NAME"),
    os.Getenv("DB_SSLMODE"),
)
```

## 🚀 Deployment Commands

### Deploy Database
```bash
# Deploy with Pulumi
cd pulumi
pulumi up

# Verify deployment
kubectl get statefulsets -n kellogg-music-match
kubectl get pods -n kellogg-music-match -l component=database
kubectl get pvc -n kellogg-music-match
```

### Database Access

#### Port Forward for Administration
```bash
# Forward PostgreSQL port for local access
kubectl port-forward -n kellogg-music-match service/postgres 5432:5432

# Connect with psql
psql -h localhost -p 5432 -U kellogg_user -d kellogg_music_match
```

#### Direct Pod Access
```bash
# Get PostgreSQL pod name
kubectl get pods -n kellogg-music-match -l component=database

# Execute psql in pod
kubectl exec -it -n kellogg-music-match postgres-0 -- psql -U kellogg_user -d kellogg_music_match
```

## 📊 Database Schema (Future Migration)

### Planned Tables

#### Users Table
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### Artists Table
```sql
CREATE TABLE artists (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### User Artists Table
```sql
CREATE TABLE user_artists (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    artist_id INTEGER REFERENCES artists(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, artist_id)
);
```

#### Indexes
```sql
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_artists_name ON artists(name);
CREATE INDEX idx_user_artists_user_id ON user_artists(user_id);
CREATE INDEX idx_user_artists_artist_id ON user_artists(artist_id);
```

## 🔧 Configuration Management

### Security Settings
- **Password**: Stored in Kubernetes Secret
- **Network**: Cluster-internal access only
- **SSL**: Disabled for development (enable for production)
- **Authentication**: Password-based authentication

### Resource Allocation
- **CPU Request**: 100m
- **Memory Request**: 256Mi
- **CPU Limit**: 500m
- **Memory Limit**: 1Gi
- **Storage**: 10Gi persistent volume

### Health Checks
- **Liveness Probe**: `pg_isready` every 10 seconds
- **Readiness Probe**: `pg_isready` every 5 seconds
- **Initial Delay**: 30 seconds (liveness), 5 seconds (readiness)

## 🛠️ Maintenance Operations

### Backup and Restore
```bash
# Backup database
kubectl exec -it -n kellogg-music-match postgres-0 -- pg_dump -U kellogg_user kellogg_music_match > backup.sql

# Restore database  
kubectl exec -i -n kellogg-music-match postgres-0 -- psql -U kellogg_user kellogg_music_match < backup.sql
```

### Database Monitoring
```bash
# Check database logs
kubectl logs -n kellogg-music-match postgres-0

# Monitor resource usage
kubectl top pod -n kellogg-music-match postgres-0

# Check persistent volume status
kubectl get pv,pvc -n kellogg-music-match
```

### Scaling Considerations
```bash
# PostgreSQL StatefulSet is configured for single replica
# For high availability, consider:
# 1. PostgreSQL streaming replication
# 2. Patroni for automated failover
# 3. External managed database services
```

## 🔄 Migration Path

### Phase 1: Infrastructure Ready ✅
- PostgreSQL StatefulSet deployed
- Backend environment variables configured
- Database connection ready for application code

### Phase 2: Code Migration (Future)
- Replace in-memory store with PostgreSQL driver
- Implement database schema creation
- Add data access layer (repository pattern)
- Update business logic to use database

### Phase 3: Production Hardening (Future)
- Enable SSL/TLS encryption
- Implement connection pooling
- Add monitoring and alerting
- Set up automated backups
- Configure high availability

## 📚 References

- [PostgreSQL Docker Official Image](https://hub.docker.com/_/postgres)
- [Kubernetes StatefulSets](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Go PostgreSQL Driver (pq)](https://github.com/lib/pq)

---

🗄️ **Database infrastructure ready for seamless migration from in-memory to persistent storage!** 🗄️
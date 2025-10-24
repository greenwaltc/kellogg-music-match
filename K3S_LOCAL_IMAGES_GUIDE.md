# K3s Local Image Development Guide

## 🎯 **Overview**

This guide shows how to set up k3s to seamlessly work with locally built Docker images, eliminating the need to constantly copy images to the cluster.

## 🚀 **Quick Start**

### **Method 1: Direct Import (Recommended for Development)**

```bash
# Build and import all images in one command
make k3s-build-import

# Or step by step:
make docker-build          # Build Docker images
make k3s-import            # Import to k3s
```

### **Method 2: Local Registry (Recommended for Teams)**

```bash
# One-time setup
make k3s-registry          # Setup local registry

# For each build cycle
make docker-build          # Build images
make k3s-push              # Push to local registry
```

## 🛠️ **Detailed Setup Options**

### **Option 1: Direct containerd Import**

This is the simplest approach for single-developer workflows:

```bash
# Build your images
docker-compose build

# Import directly into k3s containerd
sudo k3s ctr images import <(docker save kellogg-music-match-backend:latest)
sudo k3s ctr images import <(docker save kellogg-music-match-ui:latest)

# Or use our automated script
./scripts/k3s-image-import.sh import
```

**Pros:**
- ✅ Simple and fast
- ✅ No additional services needed
- ✅ Direct access to images

**Cons:**
- ❌ Manual process for each build
- ❌ Single-machine only

### **Option 2: Local Docker Registry**

Better for team environments or CI/CD:

```bash
# One-time setup: Create local registry
docker run -d --name registry --restart=always -p 5000:5000 registry:2

# Configure k3s to use local registry
sudo mkdir -p /etc/rancher/k3s
cat << EOF | sudo tee /etc/rancher/k3s/registries.yaml
mirrors:
  localhost:5000:
    endpoint:
      - "http://localhost:5000"
configs:
  "localhost:5000":
    insecure: true
EOF

# Restart k3s
sudo systemctl restart k3s

# For each build:
docker build -t localhost:5000/kellogg-music-match-backend:latest ./backend
docker push localhost:5000/kellogg-music-match-backend:latest
```

**Pros:**
- ✅ Automatic image pulling
- ✅ Works with multiple developers
- ✅ Standard Docker workflow

**Cons:**
- ❌ Additional registry service
- ❌ Network overhead

### **Option 3: Shared Docker Socket (Advanced)**

Mount Docker socket into k3s containers:

```bash
# Install k3s with docker runtime
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--docker" sh -

# Now k3s uses Docker daemon directly
# Images built with 'docker build' are immediately available
```

**Pros:**
- ✅ Zero-copy image sharing
- ✅ Automatic image availability

**Cons:**
- ❌ Security implications
- ❌ Requires Docker runtime

## 📋 **Available Commands**

```bash
# Build and import workflow
make k3s-build-import      # Build all images and import to k3s
make k3s-import            # Import existing Docker images to k3s
make docker-build          # Build Docker images only

# Registry workflow  
make k3s-registry          # Setup local Docker registry
make k3s-push              # Push images to local registry

# Monitoring and status
make k3s-images            # Show current k3s images
make k3s-status           # Show cluster and app status

# Deployment
make k3s-deploy           # Full build, import, and deploy cycle

# Direct script usage
./scripts/k3s-image-import.sh import    # Build and import
./scripts/k3s-image-import.sh registry  # Setup registry
./scripts/k3s-image-import.sh show      # Show images
./scripts/k3s-image-import.sh cleanup   # Clean unused images
```

## 🔧 **Kubernetes Manifest Updates**

Update your Kubernetes deployments to use local images:

```yaml
# For direct import method
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend
spec:
  template:
    spec:
      containers:
      - name: backend
        image: kellogg-music-match-backend:latest
        imagePullPolicy: Never  # Don't try to pull from external registry

---
# For local registry method  
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend
spec:
  template:
    spec:
      containers:
      - name: backend
        image: localhost:5000/kellogg-music-match-backend:latest
        imagePullPolicy: Always  # Always pull from local registry
```

## 🔄 **Development Workflow**

### **Daily Development Loop:**

```bash
# 1. Make code changes
vim backend/some-file.go

# 2. Build and deploy to k3s  
make k3s-build-import

# 3. Check deployment
kubectl rollout restart deployment/backend -n affyne
kubectl get pods -n affyne

# 4. Test changes
curl http://localhost:8080/health
```

### **Team Workflow with Registry:**

```bash
# One-time team setup
make k3s-registry

# Daily workflow
make docker-build
make k3s-push
kubectl rollout restart deployment/backend -n affyne
```

## 🐛 **Troubleshooting**

### **Images Not Found**
```bash
# Check if images exist in k3s
make k3s-images

# Re-import if missing
make k3s-import
```

### **Registry Connection Issues**
```bash
# Check registry is running
docker ps | grep registry

# Test registry connectivity
curl http://localhost:5000/v2/_catalog
```

### **k3s Permission Issues**
```bash
# Fix kubectl permissions
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown $USER:$USER ~/.kube/config
```

### **Pod ImagePullBackOff**
```bash
# Check pod events
kubectl describe pod <pod-name> -n affyne

# Common fixes:
kubectl patch deployment <deployment> -n affyne -p '{"spec":{"template":{"spec":{"containers":[{"name":"<container>","imagePullPolicy":"Never"}]}}}}'
```

## 🎯 **Best Practices**

1. **Use Direct Import for Solo Development**
   - Fastest feedback loop
   - Simplest setup

2. **Use Local Registry for Teams**
   - Consistent workflow
   - Better CI/CD integration

3. **Set imagePullPolicy Correctly**
   - `Never` for direct import
   - `Always` for local registry

4. **Tag Images Consistently**
   - Use `latest` for development
   - Use specific versions for production

5. **Automate with Makefile**
   - `make k3s-build-import` for complete cycle
   - `make k3s-deploy` for deployment

## 📊 **Performance Comparison**

| Method | Build Time | Deploy Time | Network | Complexity |
|--------|------------|-------------|---------|------------|
| Direct Import | Fast | Fastest | None | Low |
| Local Registry | Fast | Fast | Minimal | Medium |
| Remote Registry | Fast | Slow | High | High |

---

## 🚀 **Quick Reference**

```bash
# Complete development cycle
make k3s-build-import && kubectl rollout restart deployment/backend -n affyne

# Check everything is working
make k3s-status

# Clean up when needed
./scripts/k3s-image-import.sh cleanup
```

Your k3s cluster is now optimized for local development with zero image copying overhead! 🎉
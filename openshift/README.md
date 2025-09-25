# OpenShift Deployment Instructions for WhatsApp Go MCP Server

## Prerequisites
- OpenShift CLI (`oc`) installed and configured
- Access to an OpenShift cluster
- Proper permissions to create resources in the target namespace

## Deployment Steps

### 1. Create a new project (optional)
```bash
oc new-project whatsapp-mcp
```

### 2. Deploy all resources
```bash
# Apply all OpenShift resources
oc apply -f openshift/

# Or apply individually:
oc apply -f openshift/imagestream.yaml
oc apply -f openshift/buildconfig.yaml
oc apply -f openshift/pvc.yaml
oc apply -f openshift/deployment.yaml
oc apply -f openshift/service.yaml
oc apply -f openshift/route.yaml
```

### 3. Start the build
```bash
oc start-build whatsapp-mcp-server --follow
```

### 4. Check deployment status
```bash
# Check build status
oc get builds

# Check deployment status
oc get deployments

# Check pods
oc get pods

# Check service
oc get services

# Check route
oc get routes
```

### 5. Access the application
```bash
# Get the route URL
oc get route whatsapp-mcp-server -o jsonpath='{.spec.host}'

# Or get all route information
oc describe route whatsapp-mcp-server
```

## Configuration

### Environment Variables
The deployment includes the following environment variables:
- `PORT`: Server port (default: 8080)
- `LOG_LEVEL`: Logging level (default: info)

### Persistent Storage
- `whatsapp-data-pvc`: 1Gi storage for database files
- `whatsapp-media-pvc`: 5Gi storage for media files

### Resource Limits
- Memory: 256Mi request, 512Mi limit
- CPU: 100m request, 500m limit

## Monitoring

### Health Checks
- Liveness probe: `/health` endpoint
- Readiness probe: `/health` endpoint

### Logs
```bash
# View application logs
oc logs -f deployment/whatsapp-mcp-server

# View logs for specific pod
oc logs -f <pod-name>
```

## Troubleshooting

### Common Issues
1. **Build fails**: Check Dockerfile and source code
2. **Pod not starting**: Check resource limits and environment variables
3. **Service not accessible**: Check service and route configuration

### Useful Commands
```bash
# Describe resources for debugging
oc describe buildconfig whatsapp-mcp-server
oc describe deployment whatsapp-mcp-server
oc describe service whatsapp-mcp-server
oc describe route whatsapp-mcp-server

# Check events
oc get events --sort-by='.lastTimestamp'

# Scale deployment
oc scale deployment whatsapp-mcp-server --replicas=2
```

## Cleanup
```bash
# Delete all resources
oc delete -f openshift/

# Or delete individually
oc delete buildconfig whatsapp-mcp-server
oc delete deployment whatsapp-mcp-server
oc delete service whatsapp-mcp-server
oc delete route whatsapp-mcp-server
oc delete pvc whatsapp-data-pvc whatsapp-media-pvc
oc delete imagestream whatsapp-mcp-server
```

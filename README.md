 # compose
 
**Run Helm charts on Docker/Podman without Kubernetes**

<img width="455" height="549" alt="Generated_Image_September_28__2025_-_12_05AM-removebg-preview" src="https://github.com/user-attachments/assets/a149220d-655a-4fcf-b366-4ac70c738b09" />

`compose` is an open source tool that lets you take Helm charts and run them directly on Docker or Podman, without needing a Kubernetes cluster. It bridges the gap between Kubernetes-native packaging (Helm) and traditional container runtimes (`docker-compose.yaml`), giving you the same artifacts and deployment flows across both worlds.

## Why compose?

In most environments we prefer k3s + ArgoCD for continuous deployment and day-2 updates. But not all clients allow this — especially those running RHEL with Podman and no Kubernetes. That forces teams back to `docker-compose.yaml`, often rebuilt manually → error-prone, slow, and inconsistent.

**compose fixes this by:** 
- Reusing the same Helm pipeline artifacts you already generate
- Converting them into `docker-compose.yaml` automatically  
- Delivering a CD-like experience on Docker/Podman runtimes

No drift, no duplicate work — just consistent deployments everywhere. 

## Features

- 🚀 **Run Helm charts on Docker or Podman** - No Kubernetes cluster required
- 🔄 **Generates `docker-compose.yaml` automatically** - From any Helm chart
- 📦 **Single static Go binary** - No external dependencies
- 🔐 **Reuses host Docker/Podman auth** - Seamless registry authentication
- 🔑 **Token refresher support** - For long-lived credentials
- ⚙️ **Works in CI/CD or standalone mode** - Flexible deployment options
- 🎯 **OCI registry support** - Pull charts from any OCI-compatible registry

## Installation

### Build from source

Requires Go 1.20+:

``` bash
git clone https://github.com/ashupednekar/compose.git
cd compose
go build -o compose ./cmd/compose
```

### Download binary

Pre-built binaries are available from the [releases page](https://github.com/ashupednekar/compose/releases).

## Usage

### Basic Usage

``` bash
# Set the manifest directory (where generated files will be stored)
export MANIFEST_DIR=/path/to/your/manifests

# Sync a Helm chart from OCI registry with custom values
compose sync -c oci://registry-1.docker.io/bitnamicharts/minio -f values.yaml

# Start the generated services
./manifests/minio/restart.sh
```

### Configuration Handling

`compose` intelligently handles Kubernetes configurations by converting them to Docker-friendly formats:

**ConfigMaps & Secrets Processing:**
- **Environment Variables**: ConfigMaps and Secrets referenced via `envFrom` or `env.valueFrom` are converted to Docker environment variables
- **File Mounts**: ConfigMaps and Secrets mounted as volumes are written as individual files and mounted into containers
- **Secret Decoding**: Base64-encoded Secret values are automatically decoded before writing to files
- **Path Preservation**: Mount paths from Kubernetes are preserved in the Docker Compose setup

**Example ConfigMap/Secret Flow:**
1. Kubernetes ConfigMap with data: `{"config.yaml": "server:\n  port: 8080"}`
2. Mounted at `/app/config/config.yaml` in pod
3. Results in: `./manifests/app-name/config.yaml` file mounted to `/app/config/config.yaml` in container

### Command Reference

#### `sync` - Convert and deploy Helm chart

``` bash
compose sync [flags]

Flags:
  -c, --chart string    Helm chart location (local path or OCI registry URL)
  -f, --values string   Values file to customize the deployment
  -h, --help           Help for sync
```

### Examples

#### Deploy MinIO from Bitnami Charts

``` bash
# Create a values file to customize the deployment
cat > minio-values.yaml << EOF
ingress:
  enabled: true
  hostname: portal.beta.bankbuddy.me
  tls: true
persistence:
  storageclass: gp3encryptretain
EOF

# Sync the chart
compose sync -c oci://registry-1.docker.io/bitnamicharts/minio -f minio-values.yaml

# Start the services
./manifests/minio/restart.sh
```

## Advanced Usage

### Working with Complex Charts

``` bash
# Deploy a chart with multiple services (like a full application stack)
compose sync -c oci://registry-1.docker.io/bitnamicharts/postgresql -f postgres-values.yaml

# This creates:
# manifests/postgresql/postgresql/docker-compose.yaml
# manifests/postgresql/postgresql-metrics/docker-compose.yaml  
# manifests/postgresql/restart.sh

# Start all services together
./manifests/postgresql/restart.sh
```

### Custom Values Files

Create comprehensive values files to customize deployments:

``` yaml
# values.yaml
replicaCount: 1
image:
  repository: myapp
  tag: "v1.2.3"
  
service:
  type: ClusterIP
  port: 80

ingress:
  enabled: true
  hostname: myapp.example.com
  tls: true

persistence:
  enabled: true
  size: 10Gi
  storageClass: ssd

env:
  DATABASE_URL: "postgres://user:pass@db:5432/myapp"
  REDIS_URL: "redis://redis:6379"
```

### Managing Lifecycle Hooks

`compose` supports Kubernetes lifecycle hooks and converts them appropriately:

- **PostStart hooks** with `exec` commands are preserved
- **PostStart hooks** with `httpGet` are converted to startup checks
- Commands and arguments from containers are maintained

## How it Works

1. **Chart Resolution**: `compose` pulls the specified Helm chart from local filesystem or OCI registry
2. **Template Rendering**: Uses Helm's templating engine to render Kubernetes manifests with your values
3. **Conversion**: Converts Kubernetes resources to Docker Compose equivalents:
   - Deployments → Services with containers
   - ConfigMaps/Secrets → Volume-mounted config files  
   - Services → Network configurations
   - Ingress → Reverse proxy configurations (where applicable)
4. **Generation**: Creates `docker-compose.yaml` files and helper scripts
5. **Orchestration**: Provides restart scripts to manage the complete application stack

## Generated Manifest Structure

After running `compose sync`, you'll get a structured output based on your chart complexity:

### Single Service Chart
```
manifests/
└── <service-name>/
    ├── docker-compose.yaml
    ├── <config-file-1>
    ├── <config-file-2>
    └── ...
```

### Multi-Service Chart
```
manifests/
└── <chart-name>/
    ├── <service-1>/
    │   ├── docker-compose.yaml
    │   ├── <config-files...>
    ├── <service-2>/
    │   ├── docker-compose.yaml  
    │   ├── <config-files...>
    └── restart.sh              # Orchestrates all services
```

### File Organization Details

**Docker Compose Files**: Each service gets its own `docker-compose.yaml` with:
- Container image and command configuration
- Environment variables from ConfigMaps/Secrets
- Volume mounts for file-based configurations
- Network configuration (defaults to host networking)
- Restart policies

**Configuration Files**: 
- Created from Kubernetes ConfigMaps and Secrets that are volume-mounted
- Placed directly in the service directory
- Automatically mounted to preserve original Kubernetes paths
- Secrets are base64-decoded before writing

**Restart Script**: For multi-service charts, provides orchestrated startup/shutdown:
``` bash
./manifests/chart-name/restart.sh
# Stops all services in order, then starts them
```

## Supported Kubernetes Resources

- **Deployments** - Converted to Docker Compose services
- **StatefulSets** - Converted to Docker Compose services with volume persistence
- **ConfigMaps** - Mounted as configuration files
- **Secrets** - Mounted as secure configuration files
- **Services** - Mapped to Docker network configurations
- **PersistentVolumeClaims** - Mapped to Docker volumes

## Environment Variables

- `MANIFEST_DIR` - Directory where generated Docker Compose files are stored (required)

## Troubleshooting

### Common Issues

**Chart pulling fails**
- Ensure you have proper authentication for private registries
- Verify the chart URL format: `oci://registry.com/namespace/chart-name`

**Services not starting**  
- Check Docker/Podman daemon is running: `docker ps`
- Verify generated docker-compose.yaml syntax

**Configuration issues**
- Check generated config files in service directories
- Verify environment variable mappings in docker-compose.yaml

**Network connectivity issues**
- Services use host networking by default for simplicity  
- Check container logs: `docker logs <container-name> -f`

### Debugging Generated Output

**Inspect generated files:**
``` bash
# Check the generated docker-compose.yaml
cat manifests/<service-name>/docker-compose.yaml

# View configuration files
ls -la manifests/<service-name>/

# Check environment variables in compose file
grep -A 10 environment manifests/<service-name>/docker-compose.yaml
```

**Manual service management:**
``` bash
# Start a specific service manually
cd manifests/<chart-name>/<service-name>
docker-compose up -d

# View logs
docker-compose logs -f

# Stop service
docker-compose down
```

### Validation

**Verify conversion accuracy:**
- Compare generated environment variables with original ConfigMaps
- Check that mounted files contain expected content
- Ensure container commands match original Kubernetes specs

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Acknowledgments

- Thanks to the Helm community for the excellent templating engine
- Inspired by the need to bridge Kubernetes and traditional container runtimes
- Built for teams who want consistency across diverse deployment environments

#!/bin/bash
set -e

echo "========================================="
echo "Building Flight3 for Google App Engine"
echo "========================================="

# Set paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="$SCRIPT_DIR"
OUTPUT_DIR="$BUILD_DIR/deploy"

echo "Project root: $PROJECT_ROOT"
echo "Build directory: $BUILD_DIR"

# Clean previous build
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# Copy app.yaml
echo "Copying app.yaml..."
cp "$BUILD_DIR/app.yaml" "$OUTPUT_DIR/"

# Copy the entire Go project structure
echo "Copying Go project files..."
cp -r "$PROJECT_ROOT/cmd" "$OUTPUT_DIR/"
cp -r "$PROJECT_ROOT/internal" "$OUTPUT_DIR/"
cp "$PROJECT_ROOT/go.mod" "$OUTPUT_DIR/"
cp "$PROJECT_ROOT/go.sum" "$OUTPUT_DIR/"

# Copy public assets
echo "Copying public assets..."
if [ -d "$PROJECT_ROOT/pb_public" ]; then
    cp -r "$PROJECT_ROOT/pb_public" "$OUTPUT_DIR/"
fi

# Create a main.go entry point for App Engine
cat > "$OUTPUT_DIR/main.go" << 'EOF'
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/darianmavgo/flight3/internal/flight"
)

func main() {
	// Set up Flight3
	go flight.Flight()

	// App Engine requires listening on PORT env var
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting Flight3 on App Engine, port %s", port)
	
	// Keep the main function alive
	select {}
}
EOF

# Create .gcloudignore
cat > "$OUTPUT_DIR/.gcloudignore" << 'EOF'
# Ignore development files
.git
.gitignore
*.log
pb_data/
logs/
test_output/
tests/

# Ignore build artifacts
bin/
builds/
*.exe
*.dmg
*.deb
*.zip
*.tar.gz

# Ignore local development
.DS_Store
*.swp
*~
EOF

# Create deployment script
cat > "$OUTPUT_DIR/deploy.sh" << 'EOF'
#!/bin/bash
set -e

echo "Deploying Flight3 to Google App Engine..."
echo ""
echo "Prerequisites:"
echo "1. Install Google Cloud SDK: https://cloud.google.com/sdk/docs/install"
echo "2. Authenticate: gcloud auth login"
echo "3. Set project: gcloud config set project YOUR_PROJECT_ID"
echo ""

# Check if gcloud is installed
if ! command -v gcloud &> /dev/null; then
    echo "Error: gcloud CLI not found"
    echo "Install from: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# Check if user is authenticated
if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q .; then
    echo "Error: Not authenticated with gcloud"
    echo "Run: gcloud auth login"
    exit 1
fi

# Check if project is set
PROJECT_ID=$(gcloud config get-value project 2>/dev/null)
if [ -z "$PROJECT_ID" ]; then
    echo "Error: No GCP project set"
    echo "Run: gcloud config set project YOUR_PROJECT_ID"
    exit 1
fi

echo "Deploying to project: $PROJECT_ID"
echo ""

# Deploy to App Engine
gcloud app deploy app.yaml --quiet

echo ""
echo "========================================="
echo "✓ Deployment complete!"
echo ""
echo "Your app is available at:"
echo "https://$PROJECT_ID.appspot.com"
echo ""
echo "View logs:"
echo "  gcloud app logs tail -s default"
echo ""
echo "Open in browser:"
echo "  gcloud app browse"
echo "========================================="
EOF

chmod +x "$OUTPUT_DIR/deploy.sh"

# Create README
cat > "$OUTPUT_DIR/README.md" << 'EOF'
# Flight3 - Google App Engine Deployment

This directory contains everything needed to deploy Flight3 to Google App Engine.

## Prerequisites

1. **Install Google Cloud SDK**
   ```bash
   # macOS
   brew install google-cloud-sdk
   
   # Or download from: https://cloud.google.com/sdk/docs/install
   ```

2. **Authenticate**
   ```bash
   gcloud auth login
   ```

3. **Create/Select GCP Project**
   ```bash
   # Create new project
   gcloud projects create YOUR_PROJECT_ID
   
   # Or select existing
   gcloud config set project YOUR_PROJECT_ID
   ```

4. **Enable App Engine**
   ```bash
   gcloud app create --region=us-central
   ```

5. **Enable required APIs**
   ```bash
   gcloud services enable appengine.googleapis.com
   ```

## Deployment

### Quick Deploy
```bash
./deploy.sh
```

### Manual Deploy
```bash
gcloud app deploy app.yaml
```

### View Your App
```bash
gcloud app browse
```

## Configuration

### app.yaml
- **Runtime**: Go 1.25
- **Instance Class**: F2 (512MB RAM, 1.2GHz CPU)
- **Scaling**: 1-10 instances based on CPU utilization
- **Health Checks**: Configured for `/api/health`

### Environment Variables
Set in `app.yaml` under `env_variables`:
- `FLIGHT3_ENV=production`
- `FLIGHT3_DATA_DIR=/tmp/pb_data`
- `FLIGHT3_LOG_DIR=/tmp/logs`

## Storage Considerations

⚠️ **Important**: App Engine instances have ephemeral filesystems. Data in `/tmp` is lost when instances restart.

### Recommended Solution: Use Google Cloud Storage

1. **Create GCS bucket**
   ```bash
   gsutil mb gs://YOUR_BUCKET_NAME
   ```

2. **Mount or access via rclone**
   Flight3 already supports rclone, which can access GCS!

3. **Configure rclone remote** for GCS in PocketBase admin UI

## Monitoring

### View Logs
```bash
# Stream logs
gcloud app logs tail -s default

# Read recent logs
gcloud app logs read --limit 100
```

### View Metrics
```bash
# Open Cloud Console
gcloud app open-console
```

### Check Instances
```bash
gcloud app instances list
```

## Costs

App Engine costs depend on:
- Instance hours (F2 = ~$0.10/hour)
- Bandwidth
- Cloud Storage (if used)

**Free tier**: 28 instance hours/day

Estimate: https://cloud.google.com/products/calculator

## Troubleshooting

**Deployment fails:**
- Check Go version matches `runtime: go125`
- Verify `go.mod` is present
- Check gcloud authentication

**App crashes:**
- View logs: `gcloud app logs tail`
- Check health endpoint: `curl YOUR_APP_URL/api/health`

**Database issues:**
- Remember: local filesystem is ephemeral
- Use Cloud SQL or configure PocketBase with Cloud Storage backend

## Custom Domain

```bash
gcloud app domain-mappings create YOUR_DOMAIN.com
```

Follow DNS instructions to verify ownership.

## Rollback

```bash
# List versions
gcloud app versions list

# Route traffic to previous version
gcloud app services set-traffic default --splits VERSION_ID=1
```

## Clean Up

```bash
# Stop all traffic to version
gcloud app versions stop VERSION_ID

# Delete version
gcloud app versions delete VERSION_ID
```
EOF

echo ""
echo "========================================="
echo "✓ Google App Engine build complete!"
echo ""
echo "Output directory: $OUTPUT_DIR"
echo ""
echo "Files created:"
ls -lh "$OUTPUT_DIR"
echo ""
echo "Next steps:"
echo "1. cd $OUTPUT_DIR"
echo "2. Set up GCP: gcloud auth login && gcloud config set project YOUR_PROJECT_ID"
echo "3. Deploy: ./deploy.sh"
echo ""
echo "Read README.md for detailed setup instructions."
echo "========================================="

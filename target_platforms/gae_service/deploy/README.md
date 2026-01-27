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

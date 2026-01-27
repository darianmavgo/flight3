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

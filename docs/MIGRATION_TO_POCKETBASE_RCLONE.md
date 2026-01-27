# Migration Guide: Rclone + PocketBase Integration

This guide explains how to migrate to the new PocketBase-managed rclone configuration system in Flight3.

## Overview

Flight3 now stores all rclone remote configurations in PocketBase collections instead of static configuration files. This provides:
- Dynamic configuration without restarts
- Web UI for managing remotes
- Versioning and audit trails
- Multi-user access control

## Step 1: Access PocketBase Admin UI

1. Start Flight3: `./flight serve`
2. Navigate to `http://localhost:8090/_/`
3. Login with default credentials:
   - Email: `admin@example.com`
   - Password: `password123`
   - **Important**: Change these immediately in production!

## Step 2: Configure Rclone Remotes

### Example: Cloudflare R2

1. Go to Collections → `rclone_remotes`
2. Click "New Record"
3. Fill in the fields:
   - **name**: `r2_main` (this will be used in banquet URLs)
   - **type**: `s3`
   - **enabled**: `true`
   - **description**: `Cloudflare R2 production bucket`
   - **config** (JSON):
     ```json
     {
       "provider": "Cloudflare",
       "access_key_id": "YOUR_ACCESS_KEY_ID",
       "secret_access_key": "YOUR_SECRET_ACCESS_KEY",
       "endpoint": "https://YOUR_ACCOUNT_ID.r2.cloudflarestorage.com",
       "acl": "private"
     }
     ```
   - **vfs_settings** (JSON, optional):
     ```json
     {
       "cache_mode": "full",
       "chunk_size": 134217728
     }
     ```

### Example: Google Cloud Storage

1. Create a new record in `rclone_remotes`
2. Fill in:
   - **name**: `gcs_data`
   - **type**: `google cloud storage`
   - **enabled**: `true`
   - **description**: `GCS bucket for datasets`
   - **config** (JSON):
     ```json
     {
       "project_number": "YOUR_PROJECT_NUMBER",
       "service_account_file": "/path/to/service-account.json",
       "bucket_policy_only": "true"
     }
     ```

### Example: AWS S3

1. Create a new record in `rclone_remotes`
2. Fill in:
   - **name**: `s3_prod`
   - **type**: `s3`
   - **enabled**: `true`
   - **description**: `AWS S3 production bucket`
   - **config** (JSON):
     ```json
     {
       "provider": "AWS",
       "access_key_id": "YOUR_ACCESS_KEY_ID",
       "secret_access_key": "YOUR_SECRET_ACCESS_KEY",
       "region": "us-east-1"
     }
     ```

### Example: Local Filesystem

1. Create a new record in `rclone_remotes`
2. Fill in:
   - **name**: `local`
   - **type**: `local`
   - **enabled**: `true`
   - **description**: `Local filesystem access`
   - **config** (JSON):
     ```json
     {
       "nounc": "true"
     }
     ```

## Step 3: Configure mksqlite Converters (Optional)

If you need custom conversion settings:

1. Go to Collections → `mksqlite_configs`
2. Create records for custom converters:
   - **name**: `csv_semicolon`
   - **driver**: `csv`
   - **args** (JSON):
     ```json
     {
       "delimiter": ";",
       "header": true
     }
     ```

## Step 4: Create Data Pipelines (Optional)

For frequently accessed datasets, create pipeline records:

1. Go to Collections → `data_pipelines`
2. Create a new record:
   - **name**: `sales_data`
   - **rclone_remote**: Select `r2_main` (or your remote)
   - **rclone_path**: `data/sales/2024.csv`
   - **mksqlite_config**: Select a config or leave empty for defaults
   - **cache_ttl**: `1440` (24 hours in minutes)

## Step 5: Test Your Configuration

### Using Banquet URLs

Once configured, access your data via banquet URLs:

```
http://localhost:8090/r2_main/data/sales/2024.csv
http://localhost:8090/gcs_data/datasets/customers.xlsx/Name,Email
http://localhost:8090/s3_prod/reports/monthly.csv?where=month='January'
```

The format is:
```
http://localhost:8090/{remote_name}/{path_to_file}[/{columns}][?query_params]
```

### Verify Cache

Check that files are being cached:
```bash
ls -la pb_data/cache/
```

You should see `.db` files with cache keys.

## Step 6: Security Considerations

### Protect Sensitive Data

1. **Change default admin password** immediately
2. **Set up access rules** in PocketBase for the `rclone_remotes` collection
3. **Use environment variables** for secrets instead of hardcoding in config JSON
4. **Enable HTTPS** in production
5. **Restrict API access** to authorized users only

### Example: Using Environment Variables

Instead of storing secrets directly in the config JSON, you can reference environment variables:

```json
{
  "access_key_id": "${R2_ACCESS_KEY}",
  "secret_access_key": "${R2_SECRET_KEY}"
}
```

Then set these in your environment before starting Flight3.

## Step 7: Monitoring and Maintenance

### Check Logs

Flight3 logs all rclone operations:
```bash
tail -f pb_data/logs/flight.log
```

Look for:
- `[RCLONE]` - VFS operations
- `[CONVERTER]` - File conversions
- `[SERVER]` - Query execution
- `[BANQUET]` - Request handling

### Clear Cache

To clear expired cache files:
```bash
find pb_data/cache -name "*.db" -mtime +1 -delete
```

### Disable a Remote

To temporarily disable a remote without deleting it:
1. Go to `rclone_remotes` collection
2. Find the remote record
3. Set `enabled` to `false`

## Troubleshooting

### "Remote not found" Error

- Check that the remote name in your URL matches the `name` field in `rclone_remotes`
- Verify the remote is `enabled`
- Check PocketBase logs for collection errors

### "Failed to create filesystem" Error

- Verify the `type` field matches a valid rclone backend
- Check that all required config fields are present
- Test credentials manually with rclone CLI

### Conversion Failures

- Ensure `mksqlite` binary is in PATH
- Check file format is supported
- Verify file is accessible via the remote

### Cache Issues

- Check disk space in `pb_data/cache/`
- Verify cache directory permissions
- Try deleting cache files and re-fetching

## Advanced Configuration

### Custom VFS Settings

For performance tuning, add `vfs_settings` to your remote:

```json
{
  "cache_mode": "full",
  "chunk_size": 268435456,
  "dir_cache_time": 600,
  "cache_max_age": 86400
}
```

### Multiple Environments

Create separate remotes for dev/staging/prod:
- `r2_dev`
- `r2_staging`
- `r2_prod`

Switch between them by changing the hostname in your banquet URLs.

## Migration Checklist

- [ ] PocketBase admin password changed
- [ ] All rclone remotes configured
- [ ] Test access to each remote
- [ ] Cache directory has sufficient space
- [ ] Logs are being written correctly
- [ ] Access rules configured for security
- [ ] Backup of PocketBase database configured
- [ ] Monitoring set up for cache size
- [ ] Documentation updated for team

## Next Steps

- Set up automated cache cleanup
- Configure backups for PocketBase data
- Implement access control rules
- Set up monitoring and alerting
- Document your specific remote configurations for your team

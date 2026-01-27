# Quick Reference: Rclone + PocketBase Integration

## Quick Start

```bash
# 1. Build
go build ./cmd/flight

# 2. Run
./flight serve

# 3. Access Admin UI
open http://localhost:8090/_/
# Login: admin@example.com / password123

# 4. Configure a remote (see examples below)

# 5. Access your data
open http://localhost:8090/your_remote/path/to/file.csv
```

## Common Remote Configurations

### Cloudflare R2
```json
{
  "name": "r2_main",
  "type": "s3",
  "enabled": true,
  "config": {
    "provider": "Cloudflare",
    "access_key_id": "YOUR_KEY",
    "secret_access_key": "YOUR_SECRET",
    "endpoint": "https://ACCOUNT_ID.r2.cloudflarestorage.com"
  }
}
```

### Google Cloud Storage
```json
{
  "name": "gcs_data",
  "type": "google cloud storage",
  "enabled": true,
  "config": {
    "project_number": "YOUR_PROJECT",
    "service_account_file": "/path/to/key.json"
  }
}
```

### AWS S3
```json
{
  "name": "s3_prod",
  "type": "s3",
  "enabled": true,
  "config": {
    "provider": "AWS",
    "access_key_id": "YOUR_KEY",
    "secret_access_key": "YOUR_SECRET",
    "region": "us-east-1"
  }
}
```

### Local Files
```json
{
  "name": "local",
  "type": "local",
  "enabled": true,
  "config": {}
}
```

## Banquet URL Examples

```bash
# Basic file access
/r2_main/data/sales.csv

# Select specific columns
/r2_main/data/sales.csv/Name,Amount,Date

# Filter with WHERE clause
/r2_main/data/sales.csv?where=Amount>1000

# Sort results
/r2_main/data/sales.csv/+Amount  # Ascending
/r2_main/data/sales.csv/-Amount  # Descending

# Limit results
/r2_main/data/sales.csv?limit=100

# Combine operations
/r2_main/data/sales.csv/Name,Amount?where=Amount>1000&limit=50
```

## Architecture Flow

```
Request → Parse → Lookup Remote → Get VFS → Check Cache
                                              ↓
                                         [Hit] → Serve
                                              ↓
                                        [Miss] → Fetch → Convert → Cache → Serve
```

## File Locations

```
pb_data/
├── cache/           # Cached SQLite databases
├── temp/            # Temporary files during conversion
└── data.db          # PocketBase database (collections)
```

## Troubleshooting

### "Remote not found"
- Check remote `name` matches URL hostname
- Verify `enabled` is true
- Check PocketBase admin UI

### "Failed to create filesystem"
- Verify `type` is valid rclone backend
- Check all required config fields present
- Test credentials with rclone CLI

### "Conversion failed"
- Ensure `mksqlite` is in PATH
- Check file format is supported
- Verify file is accessible

### Cache issues
- Check disk space
- Verify permissions on `pb_data/cache/`
- Try deleting cache files

## Useful Commands

```bash
# Check cache size
du -sh pb_data/cache/

# Clear old cache (>24 hours)
find pb_data/cache -name "*.db" -mtime +1 -delete

# View logs
tail -f pb_data/logs/*.log

# Test a remote (via rclone CLI)
rclone ls your_remote:

# Rebuild
go build ./cmd/flight

# Run tests
go test ./tests/...
```

## Security Checklist

- [ ] Change default admin password
- [ ] Configure PocketBase access rules
- [ ] Use HTTPS in production
- [ ] Store secrets in environment variables
- [ ] Restrict API access
- [ ] Enable audit logging
- [ ] Regular backups of pb_data/

## Performance Tips

1. **Cache TTL**: Set appropriate TTL per dataset
   - Frequently updated: 60-120 minutes
   - Static data: 1440 minutes (24 hours)

2. **VFS Settings**: Tune per remote
   ```json
   "vfs_settings": {
     "cache_mode": "full",
     "chunk_size": 134217728
   }
   ```

3. **Query Optimization**: Use WHERE and LIMIT
   ```
   ?where=date>'2024-01-01'&limit=1000
   ```

## Documentation Links

- [Full Architecture](RCLONE_POCKETBASE.md)
- [Migration Guide](MIGRATION_TO_POCKETBASE_RCLONE.md)
- [Implementation Plan](RCLONE_POCKETBASE_IMPLEMENTATION_PLAN.md)
- [Implementation Summary](IMPLEMENTATION_SUMMARY.md)

## Support

For issues or questions:
1. Check troubleshooting section above
2. Review full documentation
3. Check PocketBase logs
4. Verify rclone configuration

---

**Quick Reference v1.0** - Flight3 Rclone+PocketBase Integration

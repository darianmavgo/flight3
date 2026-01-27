# Flight3 for macOS Apple Silicon

This is Flight3 compiled natively for Apple Silicon Macs (M1, M2, M3, M4).

## Installation

1. Copy Flight3.app to your Applications folder
2. On first launch, you may need to right-click and select "Open" to bypass Gatekeeper
3. The app will start a local server at http://127.0.0.1:8090

## Accessing Flight3

Once the app is running:
- Admin UI: http://127.0.0.1:8090/_/
- Data API: http://127.0.0.1:8090/

## Data Location

Flight3 stores its data in:
`~/Library/Application Support/Flight3/`

This includes:
- `pb_data/` - PocketBase database and cache
- `logs/` - Application logs

## Running from Terminal

You can also run Flight3 from the command line:

```bash
/Applications/Flight3.app/Contents/MacOS/flight3 serve
```

## Uninstallation

1. Quit Flight3
2. Delete Flight3.app from Applications
3. (Optional) Remove data: `rm -rf ~/Library/Application\ Support/Flight3`

## Requirements

- macOS 11.0 (Big Sur) or later
- Apple Silicon processor (M1/M2/M3/M4)

## Support

For issues and documentation, visit:
https://github.com/darianmavgo/flight3

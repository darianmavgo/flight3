Flight3 for Windows
===================

Flight3 is a modern data serving application that integrates PocketBase, Rclone, 
Banquet, and SQLiter to provide a powerful, flexible, and user-friendly interface 
for accessing and visualizing data from various cloud storage sources.

Installation
------------

1. Extract the ZIP file to a directory of your choice (e.g., C:\Program Files\Flight3)
2. Open Command Prompt or PowerShell
3. Navigate to the installation directory
4. Run: flight3.exe serve

The server will start on http://localhost:8090

Accessing Flight3
------------------

Once the server is running:
- Admin UI: http://localhost:8090/_/
- Data API: http://localhost:8090/

Data Location
-------------

Flight3 stores its data in the same directory as the executable:
- pb_data\ - PocketBase database and cache
- logs\ - Application logs
- pb_public\ - Public web assets

Running as a Windows Service
-----------------------------

To run Flight3 as a Windows service, you can use NSSM (Non-Sucking Service Manager):

1. Download NSSM from https://nssm.cc/download
2. Extract nssm.exe to the Flight3 directory
3. Open Command Prompt as Administrator
4. Run: nssm install Flight3
5. Set the path to flight3.exe and add arguments: serve --http=0.0.0.0:8090
6. Click "Install service"
7. Start the service: nssm start Flight3

Firewall Configuration
----------------------

If you want to access Flight3 from other computers on your network:

1. Open Windows Defender Firewall
2. Click "Advanced settings"
3. Click "Inbound Rules" > "New Rule"
4. Select "Port" and click Next
5. Enter port 8090 and click Next
6. Allow the connection
7. Apply to all profiles (Domain, Private, Public) as needed
8. Give it a name like "Flight3 Server"

Uninstallation
--------------

1. Stop the Flight3 process (or service if configured)
2. Delete the Flight3 directory
3. If configured as a service: nssm remove Flight3 confirm

Requirements
------------

- Windows 7 or later (64-bit)
- No additional dependencies required (SQLite is built-in)

Support
-------

For issues and documentation, visit:
https://github.com/darianmavgo/flight3

Troubleshooting
---------------

If Flight3 fails to start:
- Check if port 8090 is already in use
- Run as Administrator
- Check logs\ directory for error messages
- Ensure pb_data\ directory has write permissions

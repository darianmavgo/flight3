
### SQLiter API (Data Querying)
**Internal REST API for client access**

Endpoints:
- `GET /sqliter/file/{banquet-path}` - Download cached SQLite file
- `GET /sqliter/sync/{banquet-path}` - Get sync metadata
- `GET /sqliter/rows?path={path}&start={offset}&end={limit}` - Query rows

I hate all this hardcoded api calls. Banquet urls are already well defined and don't need extra. 

!!!# Data Flow is Wrong

### Request Lifecycle

```
1. Client Request
   ↓
   GET s3://bucket@aws/data/sales.csv;tb0/name,amount;+date?limit=100

2. Flight3 Parses URL
   ↓
   Scheme: s3
   Host: bucket@aws
   DataSetPath: data/sales.csv
   ColumnSetPath: tb0/name,amount;+date
   Query: limit=100
-->

In this example bucket is userinfo/ credential alias/id. Not hostname. Hopefully it's just the docs wrong not the parsing in banquet.

   ColumnSetPath: tb0/name,amount;+date
   Query: limit=100

The ColumnSetPath is not a valid path. It's close to a select name, amount, +date where the those 3 fields are sorted by date asc.   again confirm that banquet is parsing this correctly and error is just in documentation. 

│                                                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │  PlutoGrid   │  │ File Browser │  │  Settings    │   │
│  │  (Table UI)  │  │    (UI)      │  │    (UI)      │   │
│  └──────────────┘  └──────────────┘  └──────────────┘   │
└──────────────────────────┬────────────────────────────────┘

At the moment there is no Settings UI. 

### The Banquet URL Split

For any Banquet URL, responsibilities split at the **ColumnSetPath**:

```
[Scheme]://[User]@[Host]/[DataSetPath]/[Table];[ColumnPath]?[Query]
└────────────────────────────────────┘ └──────────────────────┘
         FLIGHT3 HANDLES                   CLIENT HANDLES
```

Canonically DatasetPath;Table;Columns?[Query]

Familiar syntax allows for / to split DatasetPath/Table/Columns 

!!!!!
Important as much as possible Flight3 only makes sqlite available to any clients include Sqliter. In the future I want to make admin routes just routes to sqlite tables that affect settings and operation. 


!!!!
```
GET /                       → Redirect to /_/ (PocketBase admin)
GET /{banquet-path}         → Convert and cache dataset
```

GET /               Banquet URL for the serve_folder in the app_settings table of Flight3 pocketbase db.
Banquet URL is also visible to the user. 
API routes are not visible to the user, the are for workers to call as needed.

!!!!!!!!!
reroute /rclone_config to /_/rclone/ 

!!!!
I don't know what GET /sqliter/sync/{path}    → Sync metadata  does.

!!!! 
This endpoint was a failed attempt at pagination. Just remove it.
GET /sqliter/rows?path=...  → Query rows with pagination. 


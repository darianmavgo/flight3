# Top 10 Features of PocketBase

Based on the official documentation and GitHub repository, here are the top 10 standout features of PocketBase:

1.  **Real-time Database**
    PocketBase uses an embedded SQLite database (in WAL mode) that is highly performant. It supports real-time subscriptions, allowing clients to listen for changes to records (create, update, delete) instantly.

2.  **Built-in Authentication**
    It comes with a comprehensive authentication system out of the box. This includes classic email/password authentication as well as OAuth2 integration with numerous providers (Google, GitHub, Facebook, Apple, Microsoft, and many more). It also handles email verification and password resets.

3.  **Visual Admin Dashboard**
    A sleek, modern user interface is built-in to manage your data. You can create collections, edit records, configure access rules, and manage application settings without writing a single line of SQL or API code.

4.  **File Storage & Media Handling**
    PocketBase allows you to upload files and attach them to records. It supports storing files locally or on S3-compatible storage (AWS, MinIO, Wasabi, etc.). It also has built-in capabilities for generating image thumbnails on the fly.

5.  **Extensible Go Framework**
    While it works great as a standalone executable, PocketBase can be used as a Go library. This allows developers to extend its functionality, intercept requests, add custom routes, and integrate custom business logic directly into the compiled binary.

6.  **Simple REST-ish API**
    Every collection you create automatically gets a fully documented, predictable REST API. It supports filtering, sorting, pagination, and expanding relations, making it incredibly easy for frontend developers to consume.

7.  **Single Executable Deployment**
    The entire backend API, the database engine, and the admin UI are compiled into a single portable binary. This makes deployment trivial—just drop the binary on a server (Linux, Windows, or macOS) and run it. No containers or complex dependencies required.

8.  **Client SDKs**
    Official, well-maintained Client SDKs are available for **JavaScript/TypeScript** (browser, Node.js) and **Dart/Flutter**. There is also a community-driven ecosystem of SDKs for other languages (Python, Go, Swift, etc.).

9.  **Access Control Rules**
    PocketBase provides a declarative permission system. You can define Access Rules for each collection (e.g., who can list, view, create, update, or delete records) using a simple SQL-like syntax (e.g., `user.id = @request.auth.id`).

10. **Self-Hosted & Open Source**
    It is fully open-source (MIT License). You own your data and infrastructure. There is no vendor lock-in, and it can be hosted anywhere from a cheap VPS to a Raspberry Pi.

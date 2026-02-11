package flight

import (
	"github.com/pocketbase/pocketbase/core"
)

// HandleAutoLogin serves a page that performs a client-side login
// using the configured superuser credentials, then redirects to the app.
// This allows "Desktop Mode" to bypass the login screen.
func HandleAutoLogin(e *core.RequestEvent) error {
	// Destination after login. Default to root, or allow override.
	redirectPath := "/"
	if q := e.Request.URL.Query().Get("redirect"); q != "" {
		redirectPath = q
	}

	// Ensure we redirect to admin if requested, as that's where the login matters most
	if redirectPath == "admin" {
		redirectPath = "/_/"
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <title>Flight3 Auto Login</title>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; display: flex; flex-direction: column; justify-content: center; align-items: center; height: 100vh; background: #f0f0f0; margin: 0; }
        .status { background: white; padding: 30px; border-radius: 12px; box-shadow: 0 4px 15px rgba(0,0,0,0.1); text-align: center; max-width: 400px; }
        .spinner { border: 3px solid #f3f3f3; border-top: 3px solid #3498db; border-radius: 50%; width: 20px; height: 20px; animation: spin 1s linear infinite; margin: 15px auto; }
        @keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }
        h2 { margin-top: 0; color: #333; }
        p { color: #666; margin-bottom: 0; }
    </style>
</head>
<body>
    <div class="status">
        <h2>Authenticating...</h2>
        <div class="spinner"></div>
        <p id="msg">Logging in as admin@example.com</p>
    </div>
    <script>
        const email = "admin@example.com";
        const password = "password123";
        const redirectPath = "` + redirectPath + `";

        async function login() {
            try {
                // 1. Attempt login
                // In PocketBase v0.23+, admins are in the '_superusers' collection.
                const res = await fetch('/api/collections/_superusers/auth-with-password', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ identity: email, password: password })
                });

                if (!res.ok) {
                    const text = await res.text();
                    throw new Error("Login failed: " + res.status + " " + res.statusText + " - " + text);
                }

                const data = await res.json();

                if (data.token) {
                    // 2. Store token in localStorage (PocketBase format)
                    const authData = {
                        token: data.token,
                        model: data.record // 'record' is returned for collection auth
                    };
                    localStorage.setItem('pocketbase_auth', JSON.stringify(authData));
                    
                    // 3. Set cookie to prevent loop
                    document.cookie = "login_done=true; path=/; max-age=86400";

                    document.getElementById('msg').innerText = 'Success! Redirecting...';

                    // 4. Redirect
                    setTimeout(() => {
                        window.location.href = redirectPath;
                    }, 300);
                } else {
                    throw new Error("No token returned");
                }
            } catch (err) {
                console.error(err);
                document.getElementById('msg').innerText = 'Authentication Error: ' + err.message;
                document.getElementById('msg').style.color = '#e74c3c';
                document.querySelector('.spinner').style.display = 'none';

                // Add retry button
                const btn = document.createElement('button');
                btn.innerText = "Retry";
                btn.style.marginTop = "10px";
                btn.style.padding = "8px 16px";
                btn.onclick = login;
                document.querySelector('.status').appendChild(btn);
            }
        }

        // Start login process
        login();
    </script>
</body>
</html>`

	return e.HTML(200, html)
}

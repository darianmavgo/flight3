package flight

// deliberately import everything here as the primary location of orchestration.
import (
	"log"
	"os"
	"strings"

	_ "github.com/darianmavgo/mksqlite/converters/all"
	"github.com/darianmavgo/sqliter/sqliter"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func Flight() {

	// Default to "serve" command if no arguments are provided
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "serve")
	}

	// Calculate absolute path for data directory to avoid CWD ambiguity

	app := pocketbase.New()

	// Initialize SQLiter with Embedded Templates
	// We use the templates embedded in the sqliter library.
	tpl, err := sqliter.GetEmbeddedTemplates()
	if err != nil {
		log.Fatal("Failed to load embedded templates from sqliter:", err)
	}

	// Initialize TableWriter with embedded templates
	tw := sqliter.NewTableWriter(tpl, sqliter.DefaultConfig())

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// 1. Register your custom catch-all route
		se.Router.Any("{path...}", func(e *core.RequestEvent) error {
			path := e.Request.URL.Path

			// 1. DIRECT POCKETBASE PATHS
			if strings.HasPrefix(path, "/_/") || strings.HasPrefix(path, "/api/") {
				return e.Next()
			}

			// 2. COMMON WEB STANDARDS (If you want PB to handle them)
			// This prevents your BanquetRouter from needing to handle mundane files.
			if path == "/favicon.ico" || path == "/robots.txt" || path == "/sitemap.xml" {
				return e.Next()
			}

			// 3. ORCHESTRATION: Pass the request to your BanquetRouter
			// Assuming BanquetRouter.ServeHTTP(w, r) or a custom handler function
			return HandleBanquet(e, tw, tpl, true)
		})

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

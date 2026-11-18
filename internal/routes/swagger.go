package routes

import (
	"net/http"
	"strings"

	_ "hris-core-service/docs" // Register the existing generated specification.
	"hris-core-service/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/swag"
)

// SetupAPIDocsRoutes serves Scalar and the existing Swaggo JSON in development.
func SetupAPIDocsRoutes(router *gin.Engine, appEnv string) {
	if !strings.EqualFold(strings.TrimSpace(appEnv), "development") {
		return
	}

	router.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(scalarHTML))
	})
	router.GET("/swagger/doc.json", func(c *gin.Context) {
		document, err := swag.ReadDoc()
		if err != nil {
			response.ErrorResponseWithCode(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "API specification is unavailable.", nil)
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(document))
	})
}

const scalarHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Mapatu API Reference</title>
  <style>
    body { margin: 0; }
    :root { --scalar-font: system-ui, sans-serif; --scalar-font-code: ui-monospace, monospace; }
  </style>
</head>
<body>
  <div id="app"></div>
  <noscript>Enable JavaScript to view the API reference, or read <a href="/swagger/doc.json">the API specification</a>.</noscript>
  <script type="module">
    import { createApiReference } from 'https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.69.2/esm.js'
    createApiReference('#app', {
      url: '/swagger/doc.json',
      proxyUrl: '',
      agent: { disabled: true },
      telemetry: false,
      withDefaultFonts: false,
      persistAuth: false,
      hideClientButton: true,
      showDeveloperTools: 'never'
    })
  </script>
</body>
</html>`

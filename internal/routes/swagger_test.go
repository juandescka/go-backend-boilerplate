package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-openapi/spec"
)

func TestScalarDevelopmentUIAndSpecification(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	SetupAPIDocsRoutes(router, "development")
	resHTML := httptest.NewRecorder()
	router.ServeHTTP(resHTML, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if resHTML.Code != http.StatusOK || !strings.Contains(resHTML.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("Scalar page unavailable: HTTP %d", resHTML.Code)
	}
	for _, expected := range []string{
		"Mapatu API Reference", "@scalar/api-reference@1.69.2/esm.js",
		"url: '/swagger/doc.json'", "proxyUrl: ''", "agent: { disabled: true }",
		"telemetry: false", "withDefaultFonts: false", "persistAuth: false",
	} {
		if !strings.Contains(resHTML.Body.String(), expected) {
			t.Fatalf("Scalar page is missing expected configuration %q", expected)
		}
	}
	for _, path := range []string{"/swagger/index.html", "/swagger/swagger-initializer.js", "/swagger/swagger-ui-bundle.js"} {
		res := httptest.NewRecorder()
		router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
		if res.Code != http.StatusNotFound {
			t.Fatalf("removed Swagger UI path %s: got HTTP %d", path, res.Code)
		}
	}

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil))
	if res.Code != http.StatusOK || !strings.Contains(res.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("specification unavailable: HTTP %d", res.Code)
	}
	var document spec.Swagger
	if err := json.Unmarshal(res.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.Swagger != "2.0" || document.Info.Title != "Mapatu Backend API" || document.BasePath != "/" || document.Host != "" {
		t.Fatal("expected Mapatu Swagger 2.0 metadata with current-origin host and root base path")
	}
	if len(document.SecurityDefinitions) != 1 || len(document.Paths.Paths) != 6 {
		t.Fatal("specification must describe operational and authentication endpoints")
	}
	bearer := document.SecurityDefinitions["BearerAuth"]
	if bearer == nil || bearer.Type != "apiKey" || bearer.In != "header" || bearer.Name != "Authorization" {
		t.Fatal("opaque bearer header scheme missing")
	}
	login := document.Paths.Paths["/v1/auth/login"].Post
	me := document.Paths.Paths["/v1/auth/me"].Get
	if login == nil || len(login.Security) != 0 || me == nil || len(me.Security) != 1 {
		t.Fatal("public login or protected current-user contract missing")
	}
	for _, path := range []string{"/v1/auth/logout", "/v1/auth/logout-all"} {
		operation := document.Paths.Paths[path].Post
		if operation == nil || len(operation.Security) != 1 {
			t.Fatalf("missing protected POST %s", path)
		}
		if _, ok := operation.Security[0]["BearerAuth"]; !ok {
			t.Fatalf("missing bearer contract for %s", path)
		}
		for _, status := range []int{200, 401, 413, 415, 500} {
			if _, ok := operation.Responses.StatusCodeResponses[status]; !ok {
				t.Fatalf("missing status%d for %s", status, path)
			}
		}
	}
	for _, status := range []int{400, 401, 413, 415, 422, 500, 503} {
		if _, ok := login.Responses.StatusCodeResponses[status]; !ok {
			t.Fatalf("login response%d missing", status)
		}
	}
	for _, path := range []string{"/health", "/ready"} {
		operation := document.Paths.Paths[path].Get
		if operation == nil || len(operation.Security) != 0 {
			t.Fatalf("missing public GET %s", path)
		}
		response := operation.Responses.StatusCodeResponses[http.StatusOK]
		if response.Schema == nil || response.Schema.Ref.String() != "#/definitions/response.SuccessResponseBody" {
			t.Fatalf("%s: expected existing success envelope", path)
		}
	}
	response := document.Paths.Paths["/ready"].Get.Responses.StatusCodeResponses[http.StatusServiceUnavailable]
	if response.Schema == nil || response.Schema.Ref.String() != "#/definitions/response.ErrorResponseBody" || !strings.Contains(response.Description, "SERVICE_NOT_READY") {
		t.Fatal("readiness 503 must use the existing error envelope/code")
	}
	successProperty := document.Definitions["response.SuccessResponseBody"].Properties["success"]
	errorProperty := document.Definitions["response.ErrorResponseBody"].Properties["error"]
	if !successProperty.Type.Contains("boolean") || errorProperty.Ref.String() != "#/definitions/response.ErrorBody" {
		t.Fatal("generated response envelopes do not match existing response types")
	}
}

func TestAPIDocsDisabledOutsideDevelopment(t *testing.T) {
	for _, appEnv := range []string{"production", " PRODUCTION ", "staging", "test", "", "developmet"} {
		t.Run(appEnv, func(t *testing.T) {
			router := gin.New()
			SetupAPIDocsRoutes(router, appEnv)
			for _, path := range []string{"/docs", "/docs/", "/swagger/doc.json", "/swagger/index.html"} {
				res := httptest.NewRecorder()
				router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
				if res.Code != http.StatusNotFound {
					t.Fatalf("%s exposed for environment %q: HTTP %d", path, appEnv, res.Code)
				}
			}
		})
	}
}

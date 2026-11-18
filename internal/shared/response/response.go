package response

import (
	"fmt"
	"net/http"
	"net/url"

	dtos "hris-core-service/internal/handlers/dtos/responses/meta"

	"github.com/gin-gonic/gin"
)

type SuccessResponseBody struct {
	Success bool          `json:"success"`
	Message string        `json:"message"`
	Data    interface{}   `json:"data"`
	Meta    *ResponseMeta `json:"meta,omitempty"`
}

type ResponseMeta struct {
	Pagination *dtos.PaginationResponse `json:"pagination,omitempty"`
	Filters    *dtos.FilterResponse     `json:"filters,omitempty"`
	Sorting    *dtos.SortingResponse    `json:"sorting,omitempty"`
}

type ErrorResponseBody struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

const errorContextKey = "mapatu.api_error"

type errorContext struct {
	code   string
	status int
}

// ErrorCodeFromContext exposes only the code attached to the emitted error,
// without inspecting a response body or retaining its message/details.
func ErrorCodeFromContext(c *gin.Context) string {
	if c == nil || c.Writer == nil || c.Writer.Status() < http.StatusBadRequest {
		return ""
	}
	value, exists := c.Get(errorContextKey)
	metadata, ok := value.(errorContext)
	if !exists || !ok || metadata.status != c.Writer.Status() {
		return ""
	}
	return metadata.code
}

func validErrorCode(code string) bool {
	if len(code) == 0 || len(code) > 64 || code[0] < 'A' || code[0] > 'Z' {
		return false
	}
	for _, ch := range code {
		if !(ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_') {
			return false
		}
	}
	return true
}

func ErrorResponse(c *gin.Context, statusCode int, message string, details interface{}) {
	ErrorResponseWithCode(c, statusCode, errorCodeFromStatus(statusCode), message, details)
}

func ErrorResponseWithCode(c *gin.Context, statusCode int, code, message string, details interface{}) {
	written := c.Writer.Written()
	errorResponse := ErrorResponseBody{
		Error: ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	}

	c.JSON(statusCode, errorResponse)
	// A late write cannot replace the response that was already committed.
	// Keep only bounded classification, never the error payload or client input.
	if !written && statusCode >= http.StatusBadRequest && c.Writer.Status() == statusCode && validErrorCode(code) {
		c.Set(errorContextKey, errorContext{code: code, status: statusCode})
	}
}

// SuccessResponse sends a standard success response with optional pagination, filters, and sorting
func SuccessResponse(c *gin.Context, data interface{}, message string, pagination *dtos.PaginationResponse, filters *dtos.FilterResponse, sorting *dtos.SortingResponse) {
	response := SuccessResponseBody{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    buildMeta(pagination, filters, sorting),
	}

	c.JSON(http.StatusOK, response)
}

func buildMeta(pagination *dtos.PaginationResponse, filters *dtos.FilterResponse, sorting *dtos.SortingResponse) *ResponseMeta {
	if pagination == nil && filters == nil && sorting == nil {
		return nil
	}

	return &ResponseMeta{
		Pagination: pagination,
		Filters:    filters,
		Sorting:    sorting,
	}
}

func errorCodeFromStatus(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusTooManyRequests:
		return "RATE_LIMIT_EXCEEDED"
	case http.StatusUnprocessableEntity:
		return "VALIDATION_FAILED"
	case http.StatusInternalServerError:
		return "INTERNAL_SERVER_ERROR"
	default:
		return "ERROR"
	}
}

func BuildPaginationResponse(c *gin.Context, page, limit int, hasNext bool) dtos.PaginationResponse {
	var baseURL string
	var query map[string][]string

	if c != nil && c.Request != nil && c.Request.URL != nil {
		baseURL = c.Request.URL.Path
		query = c.Request.URL.Query()
	} else {
		baseURL = "/"
		query = make(map[string][]string)
	}

	// Convert map back to url.Values
	queryValues := make(url.Values)
	for k, v := range query {
		for _, val := range v {
			queryValues.Add(k, val)
		}
	}

	hasPrevious := page > 1
	var nextURL, prevURL string

	if hasNext {
		queryValues.Set("page", fmt.Sprintf("%d", page+1))
		queryValues.Set("limit", fmt.Sprintf("%d", limit))
		nextURL = fmt.Sprintf("%s?%s", baseURL, queryValues.Encode())
	}

	if hasPrevious {
		queryValues.Set("page", fmt.Sprintf("%d", page-1))
		queryValues.Set("limit", fmt.Sprintf("%d", limit))
		prevURL = fmt.Sprintf("%s?%s", baseURL, queryValues.Encode())
	}

	return dtos.PaginationResponse{
		Page:        page,
		Limit:       limit,
		HasNext:     hasNext,
		HasPrevious: hasPrevious,
		NextURL:     nextURL,
		PrevURL:     prevURL,
	}
}

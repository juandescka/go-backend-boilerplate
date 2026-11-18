package httphelper

import (
	"errors"
	"net/http"

	"hris-core-service/internal/shared/validation"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type BindType string

const (
	BindJSON  BindType = "json"
	BindQuery BindType = "query"
	BindForm  BindType = "form"
)

var validate = validator.New()

func BindAndValidate[T any](c *gin.Context, target *T, bindType BindType) error {
	var err error

	switch bindType {
	case BindJSON:
		err = c.ShouldBindJSON(target)
	case BindQuery:
		err = c.ShouldBindQuery(target)
	case BindForm:
		err = c.ShouldBind(target)
	default:
		err = c.ShouldBind(target)
	}

	if err != nil {
		// If binding failed due to validation tags
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return validation.FormatValidationErrors(target, ve)
		}
		// Non-validation binding error (e.g., JSON syntax)
		return err
	}

	// Manual validation for additional rules
	if err := validate.Struct(target); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			return validation.FormatValidationErrors(target, ve)
		}
		return err
	}

	return nil
}

// IsRequestBodyTooLarge identifies the standard-library error produced by the
// HTTP boundary's MaxBytesReader without exposing its configured size.
func IsRequestBodyTooLarge(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}

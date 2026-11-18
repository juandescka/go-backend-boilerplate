package validation

import (
	"fmt"
	"reflect"
	"strings"

	"hris-core-service/internal/shared/labels"

	"github.com/go-playground/validator/v10"
)

// ValidationErrorDetail is returned as a structured validation error
type ValidationErrorDetail struct {
	Message string            `json:"message"`
	Details map[string]string `json:"details"`
}

func (v *ValidationErrorDetail) Error() string {
	return fmt.Sprintf("%s: %v", v.Message, v.Details)
}

// buildJSONMap creates a map of StructField -> jsonKey for a struct type
func buildJSONMap(t reflect.Type) map[string]string {
	jsonMap := make(map[string]string)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonKey := field.Name
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				jsonKey = parts[0]
			}
		} else {
			jsonKey = strings.ToLower(field.Name)
		}
		jsonMap[field.Name] = jsonKey
	}
	return jsonMap
}

// buildMessage creates an error message based on tag
func buildMessage(err validator.FieldError, label string) string {
	switch err.Tag() {

	case "required":
		return fmt.Sprintf("%s wajib diisi", label)

	case "email":
		return fmt.Sprintf("%s bukan alamat email yang valid", label)

	case "min":
		return fmt.Sprintf("%s minimal %s karakter", label, err.Param())

	case "max":
		return fmt.Sprintf("%s maksimal %s karakter", label, err.Param())

	case "eqfield":
		return fmt.Sprintf("%s harus sama dengan %s", label, err.Param())

	case "gte":
		return fmt.Sprintf("%s harus lebih besar atau sama dengan %s", label, err.Param())

	case "lte":
		return fmt.Sprintf("%s harus lebih kecil atau sama dengan %s", label, err.Param())

	case "oneof":
		return fmt.Sprintf(
			"%s harus salah satu dari: %s",
			label,
			translateOneOfValues(err.Param()),
		)

	default:
		return fmt.Sprintf("%s tidak valid", label)
	}
}

// FormatValidationErrors converts validator.ValidationErrors to a structured ValidationErrorDetail
func FormatValidationErrors(structPtr interface{}, errs validator.ValidationErrors) error {
	t := reflect.TypeOf(structPtr)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Precompute jsonMap once
	jsonMap := buildJSONMap(t)
	errorMessages := make(map[string]string)

	for _, err := range errs {
		// Use StructField() for lookup in jsonMap
		fieldName := err.StructField()
		jsonKey := jsonMap[fieldName]

		// Friendly label for messages
		label := labels.Get(fieldName)

		// Build message
		message := buildMessage(err, label)

		errorMessages[jsonKey] = message
	}

	return &ValidationErrorDetail{
		Message: "Kesalahan validasi",
		Details: errorMessages,
	}
}

func translateOneOfValues(values string) string {

	parts := strings.Split(values, " ")

	var translated []string

	for _, v := range parts {
		translated = append(translated, labels.Get(v))
	}

	return strings.Join(translated, ", ")
}

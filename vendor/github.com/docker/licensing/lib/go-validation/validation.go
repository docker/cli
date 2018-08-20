package validation

import (
	"fmt"
	"reflect"

	validator "github.com/asaskevich/govalidator"
)

// ErrorKind represents a validation error type
type ErrorKind int

// ErrorKind enumeration
const (
	// ErrorGeneral represents a catchall validation error. That is,
	// an error that does not match any ErrorKind.
	ErrorGeneral ErrorKind = iota
	// ErrorEmpty represents an empty field validation error.
	ErrorEmpty
	// ErrorInvalidEmail represents an invalid email validation error.
	ErrorInvalidEmail
	// ErrorInvalidURL represents an invalid url validation error.
	ErrorInvalidURL
)

// Error represents a validation error
type Error struct {
	FieldName  string
	FieldValue interface{}
	kind       ErrorKind
}

// Error returns the string representation of the error
func (e *Error) Error() string {
	msg := e.MsgForCode()
	return fmt.Sprintf("%v invalid: %v", e.FieldName, msg)
}

// MsgForCode returns a human readable message for the given ErrorKind.
// The message may optionally include the given field value in the message.
func (e *Error) MsgForCode() string {
	switch e.kind {
	case ErrorEmpty:
		return "not provided"
	case ErrorInvalidEmail:
		return fmt.Sprintf("%v is an invalid email address", e.FieldValue)
	case ErrorInvalidURL:
		return fmt.Sprintf("%v is an invalid url", e.FieldValue)
	}

	return fmt.Sprintf("'%v' is not a valid value", e.FieldValue)
}

// Errors is a list of validation Errors
type Errors []*Error

// Validater has a single function Validate, which can be used to determine
// if the interface implementation is valid. Validate should return true if the implementation
// passes validation, false otherwise. If invalid, a list of one or more validation
// Errors should be returned.
type Validater interface {
	Validate() (bool, Errors)
}

// IsEmpty returns true if the given interface is empty, false otherwise
func IsEmpty(s interface{}) bool {
	v := reflect.ValueOf(s)

	switch v.Kind() {
	case reflect.String,
		reflect.Array:
		return v.Len() == 0
	case reflect.Map,
		reflect.Slice:
		return v.Len() == 0 || v.IsNil()
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:
		return v.Int() == 0
	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32,
		reflect.Float64:
		return v.Float() == 0
	case reflect.Interface,
		reflect.Ptr:
		return v.IsNil()
	}

	return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
}

// IsEmail returns true if the given email address is valid, false otherwise
func IsEmail(str string) bool {
	return validator.IsEmail(str)
}

// IsURL returns true if the given url is valid, false otherwise
func IsURL(str string) bool {
	return validator.IsURL(str)
}

// Matches returns true if the given string matches the given pattern, false otherwise
func Matches(str, pattern string) bool {
	return validator.Matches(str, pattern)
}

// InvalidEmpty returns an empty validation Error for the given field name
func InvalidEmpty(fieldName string) *Error {
	return &Error{
		FieldName: fieldName,
		kind:      ErrorEmpty,
	}
}

// InvalidEmail returns an invalid validation error for the given field name and value
func InvalidEmail(fieldName, fieldValue string) *Error {
	return &Error{
		FieldName:  fieldName,
		FieldValue: fieldValue,
		kind:       ErrorInvalidEmail,
	}
}

// InvalidURL returns an invalid validation error for the given field name and value
func InvalidURL(fieldName, fieldValue string) *Error {
	return &Error{
		FieldName:  fieldName,
		FieldValue: fieldValue,
		kind:       ErrorInvalidURL,
	}
}

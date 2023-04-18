package schema

import (
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/xeipuuv/gojsonschema"
)

const (
	defaultVersion = "3.11"
	versionField   = "version"
)

type portsFormatChecker struct{}

func (checker portsFormatChecker) IsFormat(_ interface{}) bool {
	// TODO: implement this
	return true
}

type durationFormatChecker struct{}

func (checker durationFormatChecker) IsFormat(input interface{}) bool {
	value, ok := input.(string)
	if !ok {
		return false
	}
	_, err := time.ParseDuration(value)
	return err == nil
}

func init() {
	gojsonschema.FormatCheckers.Add("expose", portsFormatChecker{})
	gojsonschema.FormatCheckers.Add("ports", portsFormatChecker{})
	gojsonschema.FormatCheckers.Add("duration", durationFormatChecker{})
}

// Version returns the version of the config, defaulting to the latest "3.x"
// version (3.11). If only the major version "3" is specified, it is used as
// version "3.x" and returns the default version (latest 3.x).
func Version(config map[string]interface{}) string {
	version, ok := config[versionField]
	if !ok {
		return defaultVersion
	}
	return normalizeVersion(fmt.Sprintf("%v", version))
}

func normalizeVersion(version string) string {
	switch version {
	case "", "3":
		return defaultVersion
	default:
		return version
	}
}

//go:embed data/config_schema_v*.json
var schemas embed.FS

// Validate uses the jsonschema to validate the configuration
func Validate(config map[string]interface{}, version string) error {
	version = normalizeVersion(version)
	schemaData, err := schemas.ReadFile("data/config_schema_v" + version + ".json")
	if err != nil {
		return errors.Errorf("unsupported Compose file version: %s", version)
	}

	schemaLoader := gojsonschema.NewStringLoader(string(schemaData))
	dataLoader := gojsonschema.NewGoLoader(config)

	result, err := gojsonschema.Validate(schemaLoader, dataLoader)
	if err != nil {
		return err
	}

	if !result.Valid() {
		return toError(result)
	}

	return nil
}

func toError(result *gojsonschema.Result) error {
	err := getMostSpecificError(result.Errors())
	return err
}

const (
	jsonschemaOneOf = "number_one_of"
	jsonschemaAnyOf = "number_any_of"
)

func getDescription(err validationError) string {
	switch err.parent.Type() {
	case "invalid_type":
		if expectedType, ok := err.parent.Details()["expected"].(string); ok {
			return fmt.Sprintf("must be a %s", humanReadableType(expectedType))
		}
	case jsonschemaOneOf, jsonschemaAnyOf:
		if err.child == nil {
			return err.parent.Description()
		}
		return err.child.Description()
	}
	return err.parent.Description()
}

func humanReadableType(definition string) string {
	if definition[0:1] == "[" {
		allTypes := strings.Split(definition[1:len(definition)-1], ",")
		for i, t := range allTypes {
			allTypes[i] = humanReadableType(t)
		}
		return fmt.Sprintf(
			"%s or %s",
			strings.Join(allTypes[0:len(allTypes)-1], ", "),
			allTypes[len(allTypes)-1],
		)
	}
	if definition == "object" {
		return "mapping"
	}
	if definition == "array" {
		return "list"
	}
	return definition
}

type validationError struct {
	parent gojsonschema.ResultError
	child  gojsonschema.ResultError
}

func (err validationError) Error() string {
	description := getDescription(err)
	return fmt.Sprintf("%s %s", err.parent.Field(), description)
}

func getMostSpecificError(errors []gojsonschema.ResultError) validationError {
	mostSpecificError := 0
	for i, err := range errors {
		if specificity(err) > specificity(errors[mostSpecificError]) {
			mostSpecificError = i
			continue
		}

		if specificity(err) == specificity(errors[mostSpecificError]) {
			// Invalid type errors win in a tie-breaker for most specific field name
			if err.Type() == "invalid_type" && errors[mostSpecificError].Type() != "invalid_type" {
				mostSpecificError = i
			}
		}
	}

	if mostSpecificError+1 == len(errors) {
		return validationError{parent: errors[mostSpecificError]}
	}

	switch errors[mostSpecificError].Type() {
	case "number_one_of", "number_any_of":
		return validationError{
			parent: errors[mostSpecificError],
			child:  errors[mostSpecificError+1],
		}
	default:
		return validationError{parent: errors[mostSpecificError]}
	}
}

func specificity(err gojsonschema.ResultError) int {
	return len(strings.Split(err.Field(), "."))
}

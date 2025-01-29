package cli

import (
	"errors"
	"testing"

	"github.com/docker/cli/internal"
	"gotest.tools/v3/assert"
)

func TestStatusError(t *testing.T) {
	t.Run("custom status should be returned as-is", func(t *testing.T) {
		statusError := StatusError{
			Status:     "status",
			StatusCode: 1,
		}
		assert.Equal(t, statusError.Error(), "status")
	})

	t.Run("cause error should be returned as-is", func(t *testing.T) {
		statusError := StatusError{
			Cause:      errors.New("cause"),
			StatusCode: 1,
		}
		assert.Equal(t, statusError.Error(), "cause")
	})

	t.Run("generic error-message should be generated based on the StatusCode", func(t *testing.T) {
		statusError := StatusError{
			StatusCode: 1,
		}
		assert.Equal(t, statusError.Error(), "exit status 1")
	})

	t.Run("exported StatusCodeError interface should be pinned to internal.StatusError type", func(t *testing.T) {
		internalStatusError := internal.StatusError{}
		statusError := StatusError{}
		var e StatusCodeError
		assert.Check(t, errors.As(internalStatusError, &e))
		assert.Check(t, errors.As(statusError, &e))
	})
}

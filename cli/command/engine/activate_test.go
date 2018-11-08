package engine

import (
	"fmt"
	"os"
	"testing"

	"github.com/docker/cli/internal/test"
	clitypes "github.com/docker/cli/types"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"gotest.tools/assert"
	"gotest.tools/fs"
	"gotest.tools/golden"
)

const (
	// nolint: lll
	expiredLicense = `{"key_id":"irlYm3b9fdD8hMUXjazF39im7VQSSbAm9tfHK8cKUxJt","private_key":"aH5tTRDAVJpCRS2CRetTQVXIKgWUPfoCHODhDvNPvAbz","authorization":"ewogICAicGF5bG9hZCI6ICJleUpsZUhCcGNtRjBhVzl1SWpvaU1qQXhPQzB3TXkweE9GUXdOem93TURvd01Gb2lMQ0owYjJ0bGJpSTZJbkZtTVMxMlVtRmtialp5YjFaMldXdHJlVXN4VFdKMGNGUmpXR1ozVjA4MVRWZFFTM2cwUnpJd2NIYzlJaXdpYldGNFJXNW5hVzVsY3lJNk1Td2ljMk5oYm01cGJtZEZibUZpYkdWa0lqcDBjblZsTENKc2FXTmxibk5sVkhsd1pTSTZJazltWm14cGJtVWlMQ0owYVdWeUlqb2lVSEp2WkhWamRHbHZiaUo5IiwKICAgInNpZ25hdHVyZXMiOiBbCiAgICAgIHsKICAgICAgICAgImhlYWRlciI6IHsKICAgICAgICAgICAgImp3ayI6IHsKICAgICAgICAgICAgICAgImUiOiAiQVFBQiIsCiAgICAgICAgICAgICAgICJrZXlJRCI6ICJKN0xEOjY3VlI6TDVIWjpVN0JBOjJPNEc6NEFMMzpPRjJOOkpIR0I6RUZUSDo1Q1ZROk1GRU86QUVJVCIsCiAgICAgICAgICAgICAgICJraWQiOiAiSjdMRDo2N1ZSOkw1SFo6VTdCQToyTzRHOjRBTDM6T0YyTjpKSEdCOkVGVEg6NUNWUTpNRkVPOkFFSVQiLAogICAgICAgICAgICAgICAia3R5IjogIlJTQSIsCiAgICAgICAgICAgICAgICJuIjogInlkSXktbFU3bzdQY2VZLTQtcy1DUTVPRWdDeUY4Q3hJY1FJV3VLODRwSWlaY2lZNjczMHlDWW53TFNLVGx3LVU2VUNfUVJlV1Jpb01OTkU1RHM1VFlFWGJHRzZvbG0ycWRXYkJ3Y0NnLTJVVUhfT2NCOVd1UDZnUlBIcE1GTXN4RHpXd3ZheThKVXVIZ1lVTFVwbTFJdi1tcTdscDVuUV9SeHJUMEtaUkFRVFlMRU1FZkd3bTNoTU9fZ2VMUFMtaGdLUHRJSGxrZzZfV2NveFRHb0tQNzlkX3dhSFl4R05sN1doU25laUJTeGJwYlFBS2syMWxnNzk4WGI3dlp5RUFURE1yUlI5TWVFNkFkajVISnBZM0NveVJBUENtYUtHUkNLNHVvWlNvSXUwaEZWbEtVUHliYncwMDBHTy13YTJLTjhVd2dJSW0waTVJMXVXOUdrcTR6akJ5NXpoZ3F1VVhiRzliV1BBT1lycTVRYTgxRHhHY0JsSnlIWUFwLUREUEU5VEdnNHpZbVhqSm54WnFIRWR1R3FkZXZaOFhNSTB1a2ZrR0lJMTR3VU9pTUlJSXJYbEVjQmZfNDZJOGdRV0R6eHljWmVfSkdYLUxBdWF5WHJ5clVGZWhWTlVkWlVsOXdYTmFKQi1rYUNxejVRd2FSOTNzR3ctUVNmdEQwTnZMZTdDeU9ILUU2dmc2U3RfTmVUdmd2OFluaENpWElsWjhIT2ZJd05lN3RFRl9VY3o1T2JQeWttM3R5bHJOVWp0MFZ5QW10dGFjVkkyaUdpaGNVUHJtazRsVklaN1ZEX0xTVy1pN3lvU3VydHBzUFhjZTJwS0RJbzMwbEpHaE9fM0tVbWwyU1VaQ3F6SjF5RW1LcHlzSDVIRFc5Y3NJRkNBM2RlQWpmWlV2TjdVIgogICAgICAgICAgICB9LAogICAgICAgICAgICAiYWxnIjogIlJTMjU2IgogICAgICAgICB9LAogICAgICAgICAic2lnbmF0dXJlIjogIm5saTZIdzRrbW5KcTBSUmRXaGVfbkhZS2VJLVpKenM1U0d5SUpDakh1dWtnVzhBYklpVzFZYWJJR2NqWUt0QTY4dWN6T1hyUXZreGxWQXJLSlgzMDJzN0RpbzcxTlNPRzJVcnhsSjlibDFpd0F3a3ZyTEQ2T0p5MGxGLVg4WnRabXhPVmNQZmwzcmJwZFQ0dnlnWTdNcU1QRXdmb0IxTmlWZDYyZ1cxU2NSREZZcWw3R0FVaFVKNkp4QU15VzVaOXl5YVE0NV8wd0RMUk5mRjA5YWNXeVowTjRxVS1hZjhrUTZUUWZUX05ERzNCR3pRb2V3cHlEajRiMFBHb0diOFhLdDlwekpFdEdxM3lQM25VMFFBbk90a2gwTnZac1l1UFcyUnhDT3lRNEYzVlR3UkF2eF9HSTZrMVRpYmlKNnByUWluUy16Sjh6RE8zUjBuakE3OFBwNXcxcVpaUE9BdmtzZFNSYzJDcVMtcWhpTmF5YUhOVHpVNnpyOXlOZHR2S0o1QjNST0FmNUtjYXNiWURjTnVpeXBUNk90LUtqQ2I1dmYtWVpnc2FRNzJBdFBhSU4yeUpNREZHbmEwM0hpSjMxcTJRUlp5eTZrd3RYaGtwcDhTdEdIcHYxSWRaV09SVWttb0g5SFBzSGk4SExRLTZlM0tEY2x1RUQyMTNpZnljaVhtN0YzdHdaTTNHeDd1UXR1SldHaUlTZ2Z0QW9lVjZfUmI2VThkMmZxNzZuWHYxak5nckRRcE5waEZFd2tCdGRtZHZ2THByZVVYX3BWangza1AxN3pWbXFKNmNOOWkwWUc4WHg2VmRzcUxsRXUxQ2Rhd3Q0eko1M3VHMFlKTjRnUDZwc25yUS1uM0U1aFdlMDJ3d3dBZ3F3bGlPdmd4V1RTeXJyLXY2eDI0IiwKICAgICAgICAgInByb3RlY3RlZCI6ICJleUptYjNKdFlYUk1aVzVuZEdnaU9qRTNNeXdpWm05eWJXRjBWR0ZwYkNJNkltWlJJaXdpZEdsdFpTSTZJakl3TVRjdE1EVXRNRFZVTWpFNk5UYzZNek5hSW4wIgogICAgICB9CiAgIF0KfQ=="}`
)

func TestActivateNoContainerd(t *testing.T) {
	testCli.SetContainerizedEngineClient(
		func(string) (clitypes.ContainerizedClient, error) {
			return nil, fmt.Errorf("some error")
		},
	)
	isRoot = func() bool { return true }
	cmd := newActivateCommand(testCli)
	cmd.Flags().Set("license", "invalidpath")
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	assert.ErrorContains(t, err, "unable to access local containerd")
}

func TestActivateBadLicense(t *testing.T) {
	isRoot = func() bool { return true }
	c := test.NewFakeCli(&verClient{client.Client{}, types.Version{}, nil, types.Info{}, nil})
	c.SetContainerizedEngineClient(
		func(string) (clitypes.ContainerizedClient, error) {
			return &fakeContainerizedEngineClient{}, nil
		},
	)
	cmd := newActivateCommand(c)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.Flags().Set("license", "invalidpath")
	err := cmd.Execute()
	assert.Assert(t, os.IsNotExist(err))
}

func TestActivateExpiredLicenseDryRun(t *testing.T) {
	dir := fs.NewDir(t, "license", fs.WithFile("docker.lic", expiredLicense, fs.WithMode(0644)))
	defer dir.Remove()
	filename := dir.Join("docker.lic")
	isRoot = func() bool { return true }
	c := test.NewFakeCli(&verClient{client.Client{}, types.Version{}, nil, types.Info{}, nil})
	c.SetContainerizedEngineClient(
		func(string) (clitypes.ContainerizedClient, error) {
			return &fakeContainerizedEngineClient{}, nil
		},
	)
	cmd := newActivateCommand(c)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.Flags().Set("license", filename)
	cmd.Flags().Set("display-only", "true")
	c.OutBuffer().Reset()
	err := cmd.Execute()
	assert.NilError(t, err)
	golden.Assert(t, c.OutBuffer().String(), "expired-license-display-only.golden")
}

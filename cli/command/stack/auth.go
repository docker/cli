package stack

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/stacks/pkg/types"
)

// getAuthHeaderForStack looks in the StackCreate for the first available image and uses that
// to get the auth header necessary to talk to that registry
func getAuthHeaderForStack(ctx context.Context, dockerCli command.Cli, stackCreate *types.StackCreate) (string, error) {
	// Pick an image from the stack for registry auth selection
	var image string
	for _, svc := range stackCreate.Spec.Services {
		if !strings.Contains(svc.Image, "$") {
			image = svc.Image
		} else {
			image = os.Expand(svc.Image, func(key string) string {
				for _, prop := range stackCreate.Spec.PropertyValues {
					s := strings.SplitN(prop, "=", 2)
					if len(s) != 2 {
						continue
					}
					if s[0] == key {
						return s[1]
					}
				}
				return ""
			})

		}
		break
	}
	if image == "" {
		return "", fmt.Errorf("at least one image must be defined to deploy")
	}
	return command.RetrieveAuthTokenFromImage(ctx, dockerCli, image)
}

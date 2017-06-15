// demonstration of using REST API client with cli config (push/pull with credential)

package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/docker/cli/config"
	"github.com/docker/cli/registry"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	dockerregistry "github.com/docker/docker/registry"
)

func getRefAndRegistryAuth(c client.APIClient, s string) (string, string, error) {
	distributionRef, err := reference.ParseNormalizedNamed(s)
	if err != nil {
		return "", "", err
	}
	repoInfo, err := dockerregistry.ParseRepositoryInfo(distributionRef)
	if err != nil {
		return "", "", err
	}
	if err != nil {
		return "", "", err
	}
	cfg, err := config.LoadDefaultConfigFile()
	if err != nil {
		return "", "", err
	}
	authConfig, warns, err := registry.ResolveAuthConfig(context.TODO(), c, cfg, repoInfo.Index)
	if err != nil {
		return "", "", err
	}
	for _, w := range warns {
		logrus.Warn(w)
	}
	encodedAuth, err := registry.EncodeAuthToBase64(authConfig)
	return reference.FamiliarString(distributionRef), encodedAuth, err
}

func do(c client.APIClient, op, s string) error {
	ref, encodedAuth, err := getRefAndRegistryAuth(c, s)
	if err != nil {
		return err
	}
	var responseBody io.ReadCloser
	switch op {
	case "pull":
		responseBody, err = c.ImagePull(context.TODO(), ref, types.ImagePullOptions{
			RegistryAuth: encodedAuth,
		})
	case "push":
		responseBody, err = c.ImagePush(context.TODO(), ref, types.ImagePushOptions{
			RegistryAuth: encodedAuth,
		})
	default:
		err = fmt.Errorf("unknown op: %s", op)
	}
	if err != nil {
		return err
	}
	defer responseBody.Close()
	return jsonmessage.DisplayJSONMessagesStream(responseBody, os.Stdout, os.Stdout.Fd(), true, nil)
}

func main() {
	if len(os.Args) != 3 {
		logrus.Fatalf("Usage: %s pull IMAGE", os.Args[0])
	}
	op, img := os.Args[1], os.Args[2]
	c, err := client.NewEnvClient()
	if err != nil {
		logrus.Fatal(err)
	}
	if err = do(c, op, img); err != nil {
		logrus.Fatal(err)
	}
}

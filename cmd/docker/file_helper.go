package main

import (
	"os"

	credhelpers "github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker/pkg/reexec"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
)

//nolint:gosec // ignore G101: Potential hardcoded credentials
const fileCredsHelperBinary = "docker-credential-file"

func init() {
	reexec.Register(fileCredsHelperBinary, serveFileCredHelper)
}

func serveFileCredHelper() {
	configfile := config.LoadDefaultConfigFile(os.Stderr)
	store := credentials.NewFileStore(configfile)
	credhelpers.Serve(&FileHelper{
		fileStore: store,
	})
}

var _ credhelpers.Helper = &FileHelper{}

type FileHelper struct {
	fileStore credentials.Store
}

func (f *FileHelper) Add(creds *credhelpers.Credentials) error {
	return f.fileStore.Store(types.AuthConfig{
		Username:      creds.Username,
		Password:      creds.Secret,
		ServerAddress: creds.ServerURL,
	})
}

func (f *FileHelper) Delete(serverAddress string) error {
	return f.fileStore.Erase(serverAddress)
}

func (f *FileHelper) Get(serverAddress string) (string, string, error) {
	authConfig, err := f.fileStore.Get(serverAddress)
	if err != nil {
		return "", "", err
	}

	return authConfig.Username, authConfig.Password, nil
}

func (f *FileHelper) List() (map[string]string, error) {
	creds := make(map[string]string)

	authConfig, err := f.fileStore.GetAll()
	if err != nil {
		return nil, err
	}

	for k, v := range authConfig {
		creds[k] = v.Username
	}

	return creds, nil
}

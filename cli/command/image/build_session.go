package image

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/command"
	cliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/docker/api/types/versions"
	"github.com/moby/buildkit/session"
	"github.com/pkg/errors"
)

const clientSessionRemote = "client-session"

func isSessionSupported(dockerCli command.Cli, forStream bool) bool {
	if !forStream && versions.GreaterThanOrEqualTo(dockerCli.Client().ClientVersion(), "1.39") {
		return true
	}
	return dockerCli.ServerInfo().HasExperimental && versions.GreaterThanOrEqualTo(dockerCli.Client().ClientVersion(), "1.31")
}

func trySession(dockerCli command.Cli, contextDir string, forStream bool) (*session.Session, error) {
	if !isSessionSupported(dockerCli, forStream) {
		return nil, nil
	}
	sharedKey := getBuildSharedKey(contextDir)
	s, err := session.NewSession(context.Background(), filepath.Base(contextDir), sharedKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create session")
	}
	return s, nil
}

func getBuildSharedKey(dir string) string {
	// build session is hash of build dir with node based randomness
	s := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", tryNodeIdentifier(), dir)))
	return hex.EncodeToString(s[:])
}

func tryNodeIdentifier() string {
	out := cliconfig.Dir() // return config dir as default on permission error
	if err := os.MkdirAll(cliconfig.Dir(), 0700); err == nil {
		sessionFile := filepath.Join(cliconfig.Dir(), ".buildNodeID")
		if _, err := os.Lstat(sessionFile); err != nil {
			if os.IsNotExist(err) { // create a new file with stored randomness
				b := make([]byte, 32)
				if _, err := rand.Read(b); err != nil {
					return out
				}
				if err := ioutil.WriteFile(sessionFile, []byte(hex.EncodeToString(b)), 0600); err != nil {
					return out
				}
			}
		}

		dt, err := ioutil.ReadFile(sessionFile)
		if err == nil {
			return string(dt)
		}
	}
	return out
}

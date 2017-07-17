package build

import (
	"archive/tar"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/builder/remotecontext/git"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/docker/docker/pkg/pools"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/pkg/stringid"
	"github.com/pkg/errors"
)

const (
	// DefaultDockerfileName is the Default filename with Docker commands, read by docker build
	DefaultDockerfileName string = "Dockerfile"
	// archiveHeaderSize is the number of bytes in an archive header
	archiveHeaderSize = 512
)

// ValidateContextDirectory checks if all the contents of the directory
// can be read and returns an error if some files can't be read
// symlinks which point to non-existing files don't trigger an error
func ValidateContextDirectory(srcPath string, excludes []string) error {
	contextRoot, err := getContextRoot(srcPath)
	if err != nil {
		return err
	}
	return filepath.Walk(contextRoot, func(filePath string, f os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return errors.Errorf("can't stat '%s'", filePath)
			}
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		// skip this directory/file if it's not in the path, it won't get added to the context
		if relFilePath, err := filepath.Rel(contextRoot, filePath); err != nil {
			return err
		} else if skip, err := fileutils.Matches(relFilePath, excludes); err != nil {
			return err
		} else if skip {
			if f.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// skip checking if symlinks point to non-existing files, such symlinks can be useful
		// also skip named pipes, because they hanging on open
		if f.Mode()&(os.ModeSymlink|os.ModeNamedPipe) != 0 {
			return nil
		}

		if !f.IsDir() {
			currentFile, err := os.Open(filePath)
			if err != nil && os.IsPermission(err) {
				return errors.Errorf("no permission to read from '%s'", filePath)
			}
			currentFile.Close()
		}
		return nil
	})
}

// GetContextFromReader will read the contents of the given reader as either a
// Dockerfile or tar archive. Returns a tar archive used as a context and a
// path to the Dockerfile inside the tar.
func GetContextFromReader(r io.ReadCloser, dockerfileName string) (out io.ReadCloser, relDockerfile string, err error) {
	buf := bufio.NewReader(r)

	magic, err := buf.Peek(archiveHeaderSize)
	if err != nil && err != io.EOF {
		return nil, "", errors.Errorf("failed to peek context header from STDIN: %v", err)
	}

	if IsArchive(magic) {
		return ioutils.NewReadCloserWrapper(buf, func() error { return r.Close() }), dockerfileName, nil
	}

	if dockerfileName == "-" {
		return nil, "", errors.New("build context is not an archive")
	}

	// Input should be read as a Dockerfile.
	tmpDir, err := ioutil.TempDir("", "docker-build-context-")
	if err != nil {
		return nil, "", errors.Errorf("unable to create temporary context directory: %v", err)
	}

	f, err := os.Create(filepath.Join(tmpDir, DefaultDockerfileName))
	if err != nil {
		return nil, "", err
	}
	_, err = io.Copy(f, buf)
	if err != nil {
		f.Close()
		return nil, "", err
	}

	if err := f.Close(); err != nil {
		return nil, "", err
	}
	if err := r.Close(); err != nil {
		return nil, "", err
	}

	tar, err := archive.Tar(tmpDir, archive.Uncompressed)
	if err != nil {
		return nil, "", err
	}

	return ioutils.NewReadCloserWrapper(tar, func() error {
		err := tar.Close()
		os.RemoveAll(tmpDir)
		return err
	}), DefaultDockerfileName, nil

}

// IsArchive checks for the magic bytes of a tar or any supported compression
// algorithm.
func IsArchive(header []byte) bool {
	compression := archive.DetectCompression(header)
	if compression != archive.Uncompressed {
		return true
	}
	r := tar.NewReader(bytes.NewBuffer(header))
	_, err := r.Next()
	return err == nil
}

// GetContextFromGitURL uses a Git URL as context for a `docker build`. The
// git repo is cloned into a temporary directory used as the context directory.
// Returns the absolute path to the temporary context directory, the relative
// path of the dockerfile in that context directory, and a non-nil error on
// success.
func GetContextFromGitURL(gitURL, dockerfileName string) (string, string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", "", errors.Wrapf(err, "unable to find 'git'")
	}
	absContextDir, err := git.Clone(gitURL)
	if err != nil {
		return "", "", errors.Wrapf(err, "unable to 'git clone' to temporary context directory")
	}

	absContextDir, err = ResolveAndValidateContextPath(absContextDir)
	if err != nil {
		return "", "", err
	}
	relDockerfile, err := getDockerfileRelPath(absContextDir, dockerfileName)
	return absContextDir, relDockerfile, err
}

// GetContextFromURL uses a remote URL as context for a `docker build`. The
// remote resource is downloaded as either a Dockerfile or a tar archive.
// Returns the tar archive used for the context and a path of the
// dockerfile inside the tar.
func GetContextFromURL(out io.Writer, remoteURL, dockerfileName string) (io.ReadCloser, string, error) {
	response, err := getWithStatusError(remoteURL)
	if err != nil {
		return nil, "", errors.Errorf("unable to download remote context %s: %v", remoteURL, err)
	}
	progressOutput := streamformatter.NewProgressOutput(out)

	// Pass the response body through a progress reader.
	progReader := progress.NewProgressReader(response.Body, progressOutput, response.ContentLength, "", fmt.Sprintf("Downloading build context from remote url: %s", remoteURL))

	return GetContextFromReader(ioutils.NewReadCloserWrapper(progReader, func() error { return response.Body.Close() }), dockerfileName)
}

// getWithStatusError does an http.Get() and returns an error if the
// status code is 4xx or 5xx.
func getWithStatusError(url string) (resp *http.Response, err error) {
	if resp, err = http.Get(url); err != nil {
		return nil, err
	}
	if resp.StatusCode < 400 {
		return resp, nil
	}
	msg := fmt.Sprintf("failed to GET %s with status %s", url, resp.Status)
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, errors.Wrapf(err, msg+": error reading body")
	}
	return nil, errors.Errorf(msg+": %s", bytes.TrimSpace(body))
}

// GetContextFromLocalDir uses the given local directory as context for a
// `docker build`. Returns the absolute path to the local context directory,
// the relative path of the dockerfile in that context directory, and a non-nil
// error on success.
func GetContextFromLocalDir(localDir, dockerfileName string) (string, string, error) {
	localDir, err := ResolveAndValidateContextPath(localDir)
	if err != nil {
		return "", "", err
	}

	// When using a local context directory, and the Dockerfile is specified
	// with the `-f/--file` option then it is considered relative to the
	// current directory and not the context directory.
	if dockerfileName != "" && dockerfileName != "-" {
		if dockerfileName, err = filepath.Abs(dockerfileName); err != nil {
			return "", "", errors.Errorf("unable to get absolute path to Dockerfile: %v", err)
		}
	}

	relDockerfile, err := getDockerfileRelPath(localDir, dockerfileName)
	return localDir, relDockerfile, err
}

// ResolveAndValidateContextPath uses the given context directory for a `docker build`
// and returns the absolute path to the context directory.
func ResolveAndValidateContextPath(givenContextDir string) (string, error) {
	absContextDir, err := filepath.Abs(givenContextDir)
	if err != nil {
		return "", errors.Errorf("unable to get absolute context directory of given context directory %q: %v", givenContextDir, err)
	}

	// The context dir might be a symbolic link, so follow it to the actual
	// target directory.
	//
	// FIXME. We use isUNC (always false on non-Windows platforms) to workaround
	// an issue in golang. On Windows, EvalSymLinks does not work on UNC file
	// paths (those starting with \\). This hack means that when using links
	// on UNC paths, they will not be followed.
	if !isUNC(absContextDir) {
		absContextDir, err = filepath.EvalSymlinks(absContextDir)
		if err != nil {
			return "", errors.Errorf("unable to evaluate symlinks in context path: %v", err)
		}
	}

	stat, err := os.Lstat(absContextDir)
	if err != nil {
		return "", errors.Errorf("unable to stat context directory %q: %v", absContextDir, err)
	}

	if !stat.IsDir() {
		return "", errors.Errorf("context must be a directory: %s", absContextDir)
	}
	return absContextDir, err
}

// getDockerfileRelPath returns the dockerfile path relative to the context
// directory
func getDockerfileRelPath(absContextDir, givenDockerfile string) (string, error) {
	var err error

	if givenDockerfile == "-" {
		return givenDockerfile, nil
	}

	absDockerfile := givenDockerfile
	if absDockerfile == "" {
		// No -f/--file was specified so use the default relative to the
		// context directory.
		absDockerfile = filepath.Join(absContextDir, DefaultDockerfileName)

		// Just to be nice ;-) look for 'dockerfile' too but only
		// use it if we found it, otherwise ignore this check
		if _, err = os.Lstat(absDockerfile); os.IsNotExist(err) {
			altPath := filepath.Join(absContextDir, strings.ToLower(DefaultDockerfileName))
			if _, err = os.Lstat(altPath); err == nil {
				absDockerfile = altPath
			}
		}
	}

	// If not already an absolute path, the Dockerfile path should be joined to
	// the base directory.
	if !filepath.IsAbs(absDockerfile) {
		absDockerfile = filepath.Join(absContextDir, absDockerfile)
	}

	// Evaluate symlinks in the path to the Dockerfile too.
	//
	// FIXME. We use isUNC (always false on non-Windows platforms) to workaround
	// an issue in golang. On Windows, EvalSymLinks does not work on UNC file
	// paths (those starting with \\). This hack means that when using links
	// on UNC paths, they will not be followed.
	if !isUNC(absDockerfile) {
		absDockerfile, err = filepath.EvalSymlinks(absDockerfile)
		if err != nil {
			return "", errors.Errorf("unable to evaluate symlinks in Dockerfile path: %v", err)

		}
	}

	if _, err := os.Lstat(absDockerfile); err != nil {
		if os.IsNotExist(err) {
			return "", errors.Errorf("Cannot locate Dockerfile: %q", absDockerfile)
		}
		return "", errors.Errorf("unable to stat Dockerfile: %v", err)
	}

	relDockerfile, err := filepath.Rel(absContextDir, absDockerfile)
	if err != nil {
		return "", errors.Errorf("unable to get relative Dockerfile path: %v", err)
	}

	if strings.HasPrefix(relDockerfile, ".."+string(filepath.Separator)) {
		return "", errors.Errorf("the Dockerfile (%s) must be within the build context", givenDockerfile)
	}

	return relDockerfile, nil
}

// isUNC returns true if the path is UNC (one starting \\). It always returns
// false on Linux.
func isUNC(path string) bool {
	return runtime.GOOS == "windows" && strings.HasPrefix(path, `\\`)
}

// AddDockerfileToBuildContext from a ReadCloser, returns a new archive and
// the relative path to the dockerfile in the context.
func AddDockerfileToBuildContext(dockerfileCtx io.ReadCloser, buildCtx io.ReadCloser) (io.ReadCloser, string, error) {
	file, err := ioutil.ReadAll(dockerfileCtx)
	dockerfileCtx.Close()
	if err != nil {
		return nil, "", err
	}
	now := time.Now()
	hdrTmpl := &tar.Header{
		Mode:       0600,
		Uid:        0,
		Gid:        0,
		ModTime:    now,
		Typeflag:   tar.TypeReg,
		AccessTime: now,
		ChangeTime: now,
	}
	randomName := ".dockerfile." + stringid.GenerateRandomID()[:20]

	buildCtx = archive.ReplaceFileTarWrapper(buildCtx, map[string]archive.TarModifierFunc{
		// Add the dockerfile with a random filename
		randomName: func(_ string, h *tar.Header, content io.Reader) (*tar.Header, []byte, error) {
			return hdrTmpl, file, nil
		},
		// Update .dockerignore to include the random filename
		".dockerignore": func(_ string, h *tar.Header, content io.Reader) (*tar.Header, []byte, error) {
			if h == nil {
				h = hdrTmpl
			}

			b := &bytes.Buffer{}
			if content != nil {
				if _, err := b.ReadFrom(content); err != nil {
					return nil, nil, err
				}
			} else {
				b.WriteString(".dockerignore")
			}
			b.WriteString("\n" + randomName + "\n")
			return h, b.Bytes(), nil
		},
	})
	return buildCtx, randomName, nil
}

// Compress the build context for sending to the API
func Compress(buildCtx io.ReadCloser) (io.ReadCloser, error) {
	pipeReader, pipeWriter := io.Pipe()

	go func() {
		compressWriter, err := archive.CompressStream(pipeWriter, archive.Gzip)
		if err != nil {
			pipeWriter.CloseWithError(err)
		}
		defer buildCtx.Close()

		if _, err := pools.Copy(compressWriter, buildCtx); err != nil {
			pipeWriter.CloseWithError(
				errors.Wrap(err, "failed to compress context"))
			compressWriter.Close()
			return
		}
		compressWriter.Close()
		pipeWriter.Close()
	}()

	return pipeReader, nil
}

package build

import (
	"archive/tar"
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/docker/cli/cli/command/image/build/internal/git"
	"github.com/moby/go-archive"
	"github.com/moby/go-archive/compression"
	"github.com/moby/moby/client/pkg/progress"
	"github.com/moby/moby/client/pkg/streamformatter"
	"github.com/moby/patternmatcher"
)

// DefaultDockerfileName is the Default filename with Docker commands, read by docker build
//
// Deprecated: this const is no longer used and will be removed in the next release.
const DefaultDockerfileName string = "Dockerfile"

const (
	// defaultDockerfileName is the Default filename with Docker commands, read by docker build
	defaultDockerfileName string = "Dockerfile"
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

	pm, err := patternmatcher.New(excludes)
	if err != nil {
		return err
	}

	return filepath.Walk(contextRoot, func(filePath string, f os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("can't stat '%s'", filePath)
			}
			if os.IsNotExist(err) {
				return fmt.Errorf("file ('%s') not found or excluded by .dockerignore", filePath)
			}
			return err
		}

		// skip this directory/file if it's not in the path, it won't get added to the context
		if relFilePath, err := filepath.Rel(contextRoot, filePath); err != nil {
			return err
		} else if skip, err := filepathMatches(pm, relFilePath); err != nil {
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
				return fmt.Errorf("no permission to read from '%s'", filePath)
			}
			_ = currentFile.Close()
		}
		return nil
	})
}

func filepathMatches(matcher *patternmatcher.PatternMatcher, file string) (bool, error) {
	file = filepath.Clean(file)
	if file == "." {
		// Don't let them exclude everything, kind of silly.
		return false, nil
	}
	return matcher.MatchesOrParentMatches(file)
}

// DetectArchiveReader detects whether the input stream is an archive or a
// Dockerfile and returns a buffered version of input, safe to consume in lieu
// of input. If an archive is detected, ok is set to true, and to false
// otherwise, in which case it is safe to assume input represents the contents
// of a Dockerfile.
//
// Deprecated: this utility was only used internally, and will be removed in the next release.
func DetectArchiveReader(input io.ReadCloser) (rc io.ReadCloser, ok bool, err error) {
	return detectArchiveReader(input)
}

// detectArchiveReader detects whether the input stream is an archive or a
// Dockerfile and returns a buffered version of input, safe to consume in lieu
// of input. If an archive is detected, ok is set to true, and to false
// otherwise, in which case it is safe to assume input represents the contents
// of a Dockerfile.
func detectArchiveReader(input io.ReadCloser) (rc io.ReadCloser, ok bool, err error) {
	buf := bufio.NewReader(input)

	magic, err := buf.Peek(archiveHeaderSize * 2)
	if err != nil && err != io.EOF {
		return nil, false, fmt.Errorf("failed to peek context header from STDIN: %w", err)
	}

	return newReadCloserWrapper(buf, func() error { return input.Close() }), isArchive(magic), nil
}

// WriteTempDockerfile writes a Dockerfile stream to a temporary file with a
// name specified by defaultDockerfileName and returns the path to the
// temporary directory containing the Dockerfile.
//
// Deprecated: this utility was only used internally, and will be removed in the next release.
func WriteTempDockerfile(rc io.ReadCloser) (dockerfileDir string, err error) {
	return writeTempDockerfile(rc)
}

// writeTempDockerfile writes a Dockerfile stream to a temporary file with a
// name specified by defaultDockerfileName and returns the path to the
// temporary directory containing the Dockerfile.
func writeTempDockerfile(rc io.ReadCloser) (dockerfileDir string, err error) {
	// err is a named return value, due to the defer call below.
	dockerfileDir, err = os.MkdirTemp("", "docker-build-tempdockerfile-")
	if err != nil {
		return "", fmt.Errorf("unable to create temporary context directory: %w", err)
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(dockerfileDir)
		}
	}()

	f, err := os.Create(filepath.Join(dockerfileDir, defaultDockerfileName))
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, rc); err != nil {
		return "", err
	}
	return dockerfileDir, rc.Close()
}

// GetContextFromReader will read the contents of the given reader as either a
// Dockerfile or tar archive. Returns a tar archive used as a context and a
// path to the Dockerfile inside the tar.
func GetContextFromReader(rc io.ReadCloser, dockerfileName string) (out io.ReadCloser, relDockerfile string, err error) {
	rc, ok, err := detectArchiveReader(rc)
	if err != nil {
		return nil, "", err
	}

	if ok {
		return rc, dockerfileName, nil
	}

	// Input should be read as a Dockerfile.

	if dockerfileName == "-" {
		return nil, "", errors.New("build context is not an archive")
	}
	if dockerfileName != "" {
		return nil, "", errors.New("ambiguous Dockerfile source: both stdin and flag correspond to Dockerfiles")
	}

	dockerfileDir, err := writeTempDockerfile(rc)
	if err != nil {
		return nil, "", err
	}

	tarArchive, err := archive.Tar(dockerfileDir, compression.None)
	if err != nil {
		return nil, "", err
	}

	return newReadCloserWrapper(tarArchive, func() error {
		err := tarArchive.Close()
		_ = os.RemoveAll(dockerfileDir)
		return err
	}), defaultDockerfileName, nil
}

// IsArchive checks for the magic bytes of a tar or any supported compression
// algorithm.
//
// Deprecated: this utility was used internally and will be removed in the next release.
func IsArchive(header []byte) bool {
	return isArchive(header)
}

// isArchive checks for the magic bytes of a tar or any supported compression
// algorithm.
func isArchive(header []byte) bool {
	if compression.Detect(header) != compression.None {
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
		return "", "", fmt.Errorf("unable to find 'git': %w", err)
	}
	absContextDir, err := git.Clone(gitURL)
	if err != nil {
		return "", "", fmt.Errorf("unable to 'git clone' to temporary context directory: %w", err)
	}

	absContextDir, err = resolveAndValidateContextPath(absContextDir)
	if err != nil {
		return "", "", err
	}
	relDockerfile, err := getDockerfileRelPath(absContextDir, dockerfileName)
	if err == nil && strings.HasPrefix(relDockerfile, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("the Dockerfile (%s) must be within the build context", dockerfileName)
	}

	return absContextDir, relDockerfile, err
}

// GetContextFromURL uses a remote URL as context for a `docker build`. The
// remote resource is downloaded as either a Dockerfile or a tar archive.
// Returns the tar archive used for the context and a path of the
// dockerfile inside the tar.
func GetContextFromURL(out io.Writer, remoteURL, dockerfileName string) (io.ReadCloser, string, error) {
	response, err := getWithStatusError(remoteURL)
	if err != nil {
		return nil, "", fmt.Errorf("unable to download remote context %s: %w", remoteURL, err)
	}
	progressOutput := streamformatter.NewProgressOutput(out)

	// Pass the response body through a progress reader.
	progReader := progress.NewProgressReader(response.Body, progressOutput, response.ContentLength, "", "Downloading build context from remote url: "+remoteURL)

	return GetContextFromReader(newReadCloserWrapper(progReader, func() error { return response.Body.Close() }), dockerfileName)
}

// getWithStatusError does an http.Get() and returns an error if the
// status code is 4xx or 5xx.
func getWithStatusError(url string) (resp *http.Response, err error) {
	//nolint:gosec // Ignore G107: Potential HTTP request made with variable url
	if resp, err = http.Get(url); err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusBadRequest {
		return resp, nil
	}
	msg := fmt.Sprintf("failed to GET %s with status %s", url, resp.Status)
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("%s: error reading body: %w", msg, err)
	}
	return nil, fmt.Errorf("%s: %s", msg, bytes.TrimSpace(body))
}

// GetContextFromLocalDir uses the given local directory as context for a
// `docker build`. Returns the absolute path to the local context directory,
// the relative path of the dockerfile in that context directory, and a non-nil
// error on success.
func GetContextFromLocalDir(localDir, dockerfileName string) (string, string, error) {
	localDir, err := resolveAndValidateContextPath(localDir)
	if err != nil {
		return "", "", err
	}

	// When using a local context directory, and the Dockerfile is specified
	// with the `-f/--file` option then it is considered relative to the
	// current directory and not the context directory.
	if dockerfileName != "" && dockerfileName != "-" {
		if dockerfileName, err = filepath.Abs(dockerfileName); err != nil {
			return "", "", fmt.Errorf("unable to get absolute path to Dockerfile: %w", err)
		}
	}

	relDockerfile, err := getDockerfileRelPath(localDir, dockerfileName)
	return localDir, relDockerfile, err
}

// ResolveAndValidateContextPath uses the given context directory for a `docker build`
// and returns the absolute path to the context directory.
//
// Deprecated: this utility was used internally and will be removed in the next
// release. Use [DetectContextType] to detect the context-type, and use
// [GetContextFromLocalDir], [GetContextFromLocalDir], [GetContextFromGitURL],
// or [GetContextFromURL] instead.
func ResolveAndValidateContextPath(givenContextDir string) (string, error) {
	return resolveAndValidateContextPath(givenContextDir)
}

// resolveAndValidateContextPath uses the given context directory for a `docker build`
// and returns the absolute path to the context directory.
func resolveAndValidateContextPath(givenContextDir string) (string, error) {
	absContextDir, err := filepath.Abs(givenContextDir)
	if err != nil {
		return "", fmt.Errorf("unable to get absolute context directory of given context directory %q: %w", givenContextDir, err)
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
			return "", fmt.Errorf("unable to evaluate symlinks in context path: %w", err)
		}
	}

	stat, err := os.Lstat(absContextDir)
	if err != nil {
		return "", fmt.Errorf("unable to stat context directory %q: %w", absContextDir, err)
	}

	if !stat.IsDir() {
		return "", fmt.Errorf("context must be a directory: %s", absContextDir)
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
		absDockerfile = filepath.Join(absContextDir, defaultDockerfileName)

		// Just to be nice ;-) look for 'dockerfile' too but only
		// use it if we found it, otherwise ignore this check
		if _, err = os.Lstat(absDockerfile); os.IsNotExist(err) {
			altPath := filepath.Join(absContextDir, strings.ToLower(defaultDockerfileName))
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
			return "", fmt.Errorf("unable to evaluate symlinks in Dockerfile path: %w", err)
		}
	}

	if _, err := os.Lstat(absDockerfile); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("cannot locate Dockerfile: %q", absDockerfile)
		}
		return "", fmt.Errorf("unable to stat Dockerfile: %w", err)
	}

	relDockerfile, err := filepath.Rel(absContextDir, absDockerfile)
	if err != nil {
		return "", fmt.Errorf("unable to get relative Dockerfile path: %w", err)
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
	file, err := io.ReadAll(dockerfileCtx)
	_ = dockerfileCtx.Close()
	if err != nil {
		return nil, "", err
	}
	now := time.Now()
	randomName := ".dockerfile." + randomSuffix()

	buildCtx = archive.ReplaceFileTarWrapper(buildCtx, map[string]archive.TarModifierFunc{
		// Add the dockerfile with a random filename
		randomName: func(_ string, _ *tar.Header, _ io.Reader) (*tar.Header, []byte, error) {
			header := &tar.Header{
				Name:       randomName,
				Mode:       0o600,
				ModTime:    now,
				Typeflag:   tar.TypeReg,
				AccessTime: now,
				ChangeTime: now,
			}
			return header, file, nil
		},
		// Update .dockerignore to include the random filename
		".dockerignore": func(_ string, h *tar.Header, content io.Reader) (*tar.Header, []byte, error) {
			if h == nil {
				h = &tar.Header{
					Name:       ".dockerignore",
					Mode:       0o600,
					ModTime:    now,
					Typeflag:   tar.TypeReg,
					AccessTime: now,
					ChangeTime: now,
				}
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

// randomSuffix returns a unique, 20-character ID consisting of a-z, 0-9.
func randomSuffix() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err) // This shouldn't happen
	}
	return hex.EncodeToString(b)[:20]
}

// Compress the build context for sending to the API
func Compress(buildCtx io.ReadCloser) (io.ReadCloser, error) {
	pipeReader, pipeWriter := io.Pipe()

	go func() {
		compressWriter, err := compression.CompressStream(pipeWriter, archive.Gzip)
		if err != nil {
			_ = pipeWriter.CloseWithError(err)
		}
		defer func() {
			_ = buildCtx.Close()
		}()

		if _, err := io.Copy(compressWriter, buildCtx); err != nil {
			_ = pipeWriter.CloseWithError(fmt.Errorf("failed to compress context: %w", err))
			_ = compressWriter.Close()
			return
		}
		_ = compressWriter.Close()
		_ = pipeWriter.Close()
	}()

	return pipeReader, nil
}

// readCloserWrapper wraps an io.Reader, and implements an io.ReadCloser
// It calls the given callback function when closed. It should be constructed
// with [newReadCloserWrapper].
type readCloserWrapper struct {
	io.Reader
	closer func() error
}

// Close calls back the passed closer function
func (r *readCloserWrapper) Close() error {
	return r.closer()
}

// newReadCloserWrapper wraps an io.Reader, and implements an io.ReadCloser.
// It calls the given callback function when closed.
func newReadCloserWrapper(r io.Reader, closer func() error) io.ReadCloser {
	return &readCloserWrapper{
		Reader: r,
		closer: closer,
	}
}

package build

import (
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
)

// IDFile manages a file which stores the imageID created by a build
type IDFile struct {
	filename string
}

// NewIDFile returns a new IDFile
func NewIDFile(filename string) *IDFile {
	return &IDFile{filename: filename}
}

// Remove the file
func (f *IDFile) Remove() error {
	if f.filename == "" {
		return nil
	}
	if err := os.Remove(f.filename); err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "failed to remove image ID file")
	}
	return nil
}

// Save the imageID to the file
func (f *IDFile) Save(imageID string) error {
	if f.filename == "" {
		return nil
	}
	if imageID == "" {
		return errors.Errorf("server did not provide an image ID. Cannot write %s", f.filename)
	}
	return ioutil.WriteFile(f.filename, []byte(imageID), 0666)
}

package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rayfish/teacli/modules/errors"
)

// writeAtomicFile replaces regular files after a complete write. Devices and
// pipes accept streamed output without replacing the destination itself.
func writeAtomicFile(path string, write func(io.Writer) error) error {
	mode := os.FileMode(0600)
	if info, err := os.Stat(path); err == nil {
		if !info.Mode().IsRegular() {
			file, err := os.OpenFile(path, os.O_WRONLY, 0)
			if err != nil {
				return errors.NewGeneralError(fmt.Sprintf("failed to open %s: %v", path, err))
			}
			if err := write(file); err != nil {
				file.Close()
				return errors.NewGeneralError(fmt.Sprintf("failed to write %s: %v", path, err))
			}
			if err := file.Close(); err != nil {
				return errors.NewGeneralError(fmt.Sprintf("failed to close %s: %v", path, err))
			}
			return nil
		}
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return errors.NewGeneralError(fmt.Sprintf("failed to inspect %s: %v", path, err))
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".teacli-output-*")
	if err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to create output for %s: %v", path, err))
	}
	defer os.Remove(file.Name())
	if err := write(file); err != nil {
		file.Close()
		return errors.NewGeneralError(fmt.Sprintf("failed to write %s: %v", path, err))
	}
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return errors.NewGeneralError(fmt.Sprintf("failed to set permissions for %s: %v", path, err))
	}
	if err := file.Close(); err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to close output for %s: %v", path, err))
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to replace %s: %v", path, err))
	}
	return nil
}

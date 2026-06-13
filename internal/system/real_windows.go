//go:build windows

package system

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

type RealCommandRunner struct{}

func (RealCommandRunner) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

type RealFileSystem struct{}

func (RealFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (RealFileSystem) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (RealFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

func (RealFileSystem) Statfs(path string) (FsInfo, error) {
	return FsInfo{}, errors.New("statfs is not supported on windows")
}

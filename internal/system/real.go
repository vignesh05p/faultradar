package system

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
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

func (RealFileSystem) Statfs(path string) (*syscall.Statfs_t, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return nil, err
	}
	return &stat, nil
}

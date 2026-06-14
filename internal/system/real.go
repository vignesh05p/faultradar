//go:build !windows

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

func (RealFileSystem) Statfs(path string) (FsInfo, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return FsInfo{}, err
	}
	return FsInfo{
		Blocks: uint64(stat.Blocks),
		Bfree:  uint64(stat.Bfree),
		Bavail: uint64(stat.Bavail),
	}, nil
}

func (RealFileSystem) ActualSize(info os.FileInfo) int64 {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return stat.Blocks * 512
	}
	return info.Size()
}

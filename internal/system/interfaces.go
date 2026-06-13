package system

import (
	"io/fs"
	"os"
	"syscall"
)

type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
}

type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	Stat(path string) (os.FileInfo, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
	Statfs(path string) (*syscall.Statfs_t, error)
}

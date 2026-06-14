package system

import (
	"io/fs"
	"os"
)

type FsInfo struct {
	Blocks uint64
	Bfree  uint64
	Bavail uint64
}

type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
}

type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	Stat(path string) (os.FileInfo, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
	Statfs(path string) (FsInfo, error)
	ActualSize(info os.FileInfo) int64
}

package system

import (
	"errors"
	"io/fs"
	"os"
	"time"
)

type FakeCommandRunner struct {
	RunFunc func(name string, args ...string) ([]byte, error)
}

func (f FakeCommandRunner) Run(name string, args ...string) ([]byte, error) {
	if f.RunFunc != nil {
		return f.RunFunc(name, args...)
	}
	return nil, errors.New("not implemented")
}

type FakeFileSystem struct {
	ReadFileFunc   func(path string) ([]byte, error)
	StatFunc       func(path string) (os.FileInfo, error)
	WalkDirFunc    func(root string, fn fs.WalkDirFunc) error
	StatfsFunc     func(path string) (FsInfo, error)
	ActualSizeFunc func(info os.FileInfo) int64
}

func (f FakeFileSystem) ReadFile(path string) ([]byte, error) {
	if f.ReadFileFunc != nil {
		return f.ReadFileFunc(path)
	}
	return nil, os.ErrNotExist
}

func (f FakeFileSystem) Stat(path string) (os.FileInfo, error) {
	if f.StatFunc != nil {
		return f.StatFunc(path)
	}
	return nil, os.ErrNotExist
}

func (f FakeFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	if f.WalkDirFunc != nil {
		return f.WalkDirFunc(root, fn)
	}
	return os.ErrNotExist
}

func (f FakeFileSystem) Statfs(path string) (FsInfo, error) {
	if f.StatfsFunc != nil {
		return f.StatfsFunc(path)
	}
	return FsInfo{}, errors.New("not implemented")
}

func (f FakeFileSystem) ActualSize(info os.FileInfo) int64 {
	if f.ActualSizeFunc != nil {
		return f.ActualSizeFunc(info)
	}
	if ffi, ok := info.(FakeFileInfo); ok && ffi.ActualSizeVal > 0 {
		return ffi.ActualSizeVal
	}
	return info.Size()
}

type FakeFileInfo struct {
	NameVal       string
	SizeVal       int64
	IsDirVal      bool
	ActualSizeVal int64
}

func (f FakeFileInfo) Name() string       { return f.NameVal }
func (f FakeFileInfo) Size() int64        { return f.SizeVal }
func (f FakeFileInfo) Mode() os.FileMode  { return 0 }
func (f FakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f FakeFileInfo) IsDir() bool        { return f.IsDirVal }
func (f FakeFileInfo) Sys() any           { return nil }

type FakeDirEntry struct {
	NameVal string
	InfoVal os.FileInfo
	TypeVal fs.FileMode
}

func (f FakeDirEntry) Name() string               { return f.NameVal }
func (f FakeDirEntry) IsDir() bool                { return f.InfoVal.IsDir() }
func (f FakeDirEntry) Type() fs.FileMode          { return f.TypeVal }
func (f FakeDirEntry) Info() (os.FileInfo, error) { return f.InfoVal, nil }

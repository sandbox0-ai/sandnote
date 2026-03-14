//go:build darwin || linux

package fileid

import (
	"fmt"
	"os"
	"syscall"

	"github.com/sandbox0-ai/sandnote/internal/model"
)

func Read(path string) (*model.FileIdentity, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, fmt.Errorf("unexpected stat type for %s", path)
	}
	return &model.FileIdentity{
		Kind:     "posix_inode",
		DeviceID: uint64(stat.Dev),
		ObjectID: uint64(stat.Ino),
	}, nil
}

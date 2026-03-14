//go:build windows

package fileid

import (
	"syscall"

	"github.com/sandbox0-ai/sandnote/internal/model"
)

func Read(path string) (*model.FileIdentity, error) {
	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	handle, err := syscall.CreateFile(
		ptr,
		0,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(handle)

	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(handle, &info); err != nil {
		return nil, err
	}

	objectID := (uint64(info.FileIndexHigh) << 32) | uint64(info.FileIndexLow)
	return &model.FileIdentity{
		Kind:     "windows_file_id",
		DeviceID: uint64(info.VolumeSerialNumber),
		ObjectID: objectID,
	}, nil
}

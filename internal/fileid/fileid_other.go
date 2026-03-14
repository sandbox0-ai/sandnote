//go:build !darwin && !linux && !windows

package fileid

import "github.com/sandbox0-ai/sandnote/internal/model"

func Read(path string) (*model.FileIdentity, error) {
	return nil, nil
}

package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/sandbox0-ai/sandnote/internal/fileid"
	"github.com/sandbox0-ai/sandnote/internal/model"
)

var errArtifactFound = errors.New("artifact candidate found")

func prepareArtifactReference(rootPath string, artifact *model.Artifact, sourceData []byte, info os.FileInfo) {
	searchRoots := uniquePaths(
		filepath.Dir(artifact.SourceRef),
		rootPath,
	)

	artifact.ContentDigest = digestBytes(sourceData)
	artifact.Locator = &model.ArtifactLocator{
		SearchRoots:     searchRoots,
		SizeBytes:       info.Size(),
		ModTimeUnixNano: info.ModTime().UTC().UnixNano(),
		FileIdentity:    mustReadFileIdentity(artifact.SourceRef),
	}
}

func resolveArtifactReference(rootPath string, artifact model.Artifact) (model.Artifact, bool, error) {
	if artifact.IngestMode != model.ArtifactReference {
		return artifact, false, nil
	}

	currentPath, err := findArtifactPath(rootPath, artifact)
	if err != nil {
		return artifact, false, err
	}
	if currentPath == "" {
		return artifact, false, nil
	}

	updated, err := refreshReferenceArtifact(rootPath, artifact, currentPath)
	if err != nil {
		return artifact, false, err
	}

	return updated, artifact.SourceRef != updated.SourceRef || artifact.ContentDigest != updated.ContentDigest, nil
}

func refreshReferenceArtifact(rootPath string, artifact model.Artifact, path string) (model.Artifact, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return artifact, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return artifact, err
	}

	artifact.SourceRef = path
	artifact.UpdatedAt = nowUTC()
	prepareArtifactReference(rootPath, &artifact, data, info)
	return artifact, nil
}

func findArtifactPath(rootPath string, artifact model.Artifact) (string, error) {
	if pathMatchesArtifactReference(artifact, artifact.SourceRef) {
		return artifact.SourceRef, nil
	}

	for _, root := range searchRootsForArtifact(rootPath, artifact) {
		found, err := scanArtifactRoot(root, artifact)
		if err != nil {
			return "", err
		}
		if found != "" {
			return found, nil
		}
	}

	return "", nil
}

func searchRootsForArtifact(rootPath string, artifact model.Artifact) []string {
	roots := make([]string, 0, 4)
	if artifact.Locator != nil {
		roots = append(roots, artifact.Locator.SearchRoots...)
	}
	roots = append(roots, filepath.Dir(artifact.SourceRef), filepath.Clean(rootPath))
	return uniquePaths(roots...)
}

func scanArtifactRoot(root string, artifact model.Artifact) (string, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if !info.IsDir() {
		return "", nil
	}

	found := ""
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".sandnote":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Clean(path) == filepath.Clean(artifact.SourceRef) {
			return nil
		}
		if pathMatchesArtifactReference(artifact, path) {
			found = path
			return errArtifactFound
		}
		return nil
	})
	if err != nil && !errors.Is(err, errArtifactFound) {
		return "", err
	}
	return found, nil
}

func pathMatchesArtifactReference(artifact model.Artifact, path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return false
	}

	if artifact.Locator != nil && artifact.Locator.FileIdentity != nil {
		if id := mustReadFileIdentity(path); id != nil && fileIdentityEqual(*artifact.Locator.FileIdentity, *id) {
			return true
		}
	}

	if artifact.Locator != nil && artifact.Locator.SizeBytes > 0 && info.Size() != artifact.Locator.SizeBytes {
		return false
	}

	if filepath.Clean(path) == filepath.Clean(artifact.SourceRef) && (artifact.Locator == nil || artifact.Locator.FileIdentity == nil) {
		return true
	}

	if artifact.ContentDigest == "" {
		return false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return digestBytes(data) == artifact.ContentDigest
}

func fileIdentityEqual(left, right model.FileIdentity) bool {
	return left.Kind != "" &&
		left.Kind == right.Kind &&
		left.DeviceID == right.DeviceID &&
		left.ObjectID == right.ObjectID
}

func mustReadFileIdentity(path string) *model.FileIdentity {
	id, err := fileid.Read(path)
	if err != nil {
		return nil
	}
	return id
}

func uniquePaths(paths ...string) []string {
	filtered := make([]string, 0, len(paths))
	for _, path := range paths {
		path = filepath.Clean(path)
		if path == "." || path == "" {
			continue
		}
		if slices.Contains(filtered, path) {
			continue
		}
		filtered = append(filtered, path)
	}
	return filtered
}

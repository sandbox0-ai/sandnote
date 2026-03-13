package fsstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sandbox0-ai/sandnote/internal/model"
)

const markerFile = "sandnote.json"

type Store struct {
	root string
}

type Marker struct {
	Version int `json:"version"`
}

func New(root string) *Store {
	return &Store{root: root}
}

func (s *Store) Root() string {
	return s.root
}

func (s *Store) Init() error {
	if s.root == "" {
		return errors.New("store root is required")
	}
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return fmt.Errorf("create store root: %w", err)
	}
	for _, dir := range []string{"entries", "threads", "workspaces", "topics"} {
		if err := os.MkdirAll(filepath.Join(s.root, dir), 0o755); err != nil {
			return fmt.Errorf("create %s directory: %w", dir, err)
		}
	}
	return writeJSON(filepath.Join(s.root, markerFile), Marker{Version: 1})
}

func (s *Store) Initialized() bool {
	info, err := os.Stat(filepath.Join(s.root, markerFile))
	return err == nil && !info.IsDir()
}

func (s *Store) SaveEntry(entry model.Entry) error {
	if err := entry.Validate(); err != nil {
		return err
	}
	return s.save("entries", entry.ID, entry)
}

func (s *Store) LoadEntry(id string) (model.Entry, error) {
	var entry model.Entry
	err := s.load("entries", id, &entry)
	return entry, err
}

func (s *Store) SaveThread(thread model.Thread) error {
	if err := thread.Validate(); err != nil {
		return err
	}
	return s.save("threads", thread.ID, thread)
}

func (s *Store) LoadThread(id string) (model.Thread, error) {
	var thread model.Thread
	err := s.load("threads", id, &thread)
	return thread, err
}

func (s *Store) SaveWorkspace(workspace model.Workspace) error {
	if err := workspace.Validate(); err != nil {
		return err
	}
	return s.save("workspaces", workspace.ID, workspace)
}

func (s *Store) LoadWorkspace(id string) (model.Workspace, error) {
	var workspace model.Workspace
	err := s.load("workspaces", id, &workspace)
	return workspace, err
}

func (s *Store) SaveTopic(topic model.Topic) error {
	if err := topic.Validate(); err != nil {
		return err
	}
	return s.save("topics", topic.ID, topic)
}

func (s *Store) LoadTopic(id string) (model.Topic, error) {
	var topic model.Topic
	err := s.load("topics", id, &topic)
	return topic, err
}

func (s *Store) save(kind, id string, value any) error {
	if !s.Initialized() {
		return errors.New("store is not initialized")
	}
	return writeJSON(s.objectPath(kind, id), value)
}

func (s *Store) load(kind, id string, target any) error {
	if !s.Initialized() {
		return errors.New("store is not initialized")
	}
	data, err := os.ReadFile(s.objectPath(kind, id))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode %s %q: %w", kind, id, err)
	}
	return nil
}

func (s *Store) objectPath(kind, id string) string {
	return filepath.Join(s.root, kind, id+".json")
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ensure directory %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

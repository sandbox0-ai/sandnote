package fsstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/sandbox0-ai/sandnote/internal/model"
)

const markerFile = "sandnote.json"
const replSessionFile = "repl-session.json"
const derivedIndexFile = "index/index.json"

type Store struct {
	root string
}

type Marker struct {
	Version int `json:"version"`
}

type REPLSession struct {
	CurrentWorkspace         string   `json:"current_workspace,omitempty"`
	FocusThread              string   `json:"focus_thread,omitempty"`
	InspectionScope          []string `json:"inspection_scope,omitempty"`
	PendingCheckpointContext string   `json:"pending_checkpoint_context,omitempty"`
}

type DerivedIndex struct {
	GeneratedAt time.Time                `json:"generated_at"`
	Threads     []DerivedThreadRecord    `json:"threads,omitempty"`
	Workspaces  []DerivedWorkspaceRecord `json:"workspaces,omitempty"`
	Topics      []DerivedTopicRecord     `json:"topics,omitempty"`
	Artifacts   []DerivedArtifactRecord  `json:"artifacts,omitempty"`
}

type DerivedThreadRecord struct {
	ID            string              `json:"id"`
	Question      string              `json:"question"`
	Vitality      model.VitalityState `json:"vitality"`
	WorkspaceID   string              `json:"workspace_id,omitempty"`
	TopicIDs      []string            `json:"topic_ids,omitempty"`
	SupportingIDs []string            `json:"supporting_ids,omitempty"`
	CurrentBelief string              `json:"current_belief,omitempty"`
	OpenEdge      string              `json:"open_edge,omitempty"`
	NextLean      string              `json:"next_lean,omitempty"`
	ReentryAnchor string              `json:"reentry_anchor,omitempty"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

type DerivedWorkspaceRecord struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	FocusThreadID string    `json:"focus_thread_id,omitempty"`
	ThreadCount   int       `json:"thread_count"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type DerivedTopicRecord struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Orientation string    `json:"orientation,omitempty"`
	ThreadCount int       `json:"thread_count"`
	EntryCount  int       `json:"entry_count"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DerivedArtifactRecord struct {
	ID         string                   `json:"id"`
	Kind       string                   `json:"kind"`
	SourceRef  string                   `json:"source_ref"`
	IngestMode model.ArtifactIngestMode `json:"ingest_mode"`
	UpdatedAt  time.Time                `json:"updated_at"`
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
	for _, dir := range []string{"entries", "threads", "workspaces", "topics", "artifacts"} {
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

func (s *Store) SaveREPLSession(session REPLSession) error {
	if !s.Initialized() {
		return errors.New("store is not initialized")
	}
	return writeJSON(filepath.Join(s.root, replSessionFile), session)
}

func (s *Store) LoadREPLSession() (REPLSession, error) {
	if !s.Initialized() {
		return REPLSession{}, errors.New("store is not initialized")
	}
	path := filepath.Join(s.root, replSessionFile)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return REPLSession{}, nil
	} else if err != nil {
		return REPLSession{}, err
	}

	var session REPLSession
	if err := s.loadFile(path, &session); err != nil {
		return REPLSession{}, err
	}
	return session, nil
}

func (s *Store) SaveDerivedIndex(index DerivedIndex) error {
	if !s.Initialized() {
		return errors.New("store is not initialized")
	}
	return writeJSON(filepath.Join(s.root, derivedIndexFile), index)
}

func (s *Store) LoadDerivedIndex() (DerivedIndex, error) {
	if !s.Initialized() {
		return DerivedIndex{}, errors.New("store is not initialized")
	}
	path := filepath.Join(s.root, derivedIndexFile)
	if _, err := os.Stat(path); err != nil {
		return DerivedIndex{}, err
	}

	var index DerivedIndex
	if err := s.loadFile(path, &index); err != nil {
		return DerivedIndex{}, err
	}
	return index, nil
}

func (s *Store) SaveEntry(entry model.Entry) error {
	if err := entry.Validate(); err != nil {
		return err
	}
	return s.save("entries", entry.ID, entry)
}

func (s *Store) SaveArtifact(artifact model.Artifact) error {
	if err := artifact.Validate(); err != nil {
		return err
	}
	return s.save("artifacts", artifact.ID, artifact)
}

func (s *Store) LoadArtifact(id string) (model.Artifact, error) {
	var artifact model.Artifact
	err := s.load("artifacts", id, &artifact)
	return artifact, err
}

func (s *Store) ListArtifacts() ([]model.Artifact, error) {
	files, err := s.listObjectFiles("artifacts")
	if err != nil {
		return nil, err
	}

	artifacts := make([]model.Artifact, 0, len(files))
	for _, file := range files {
		var artifact model.Artifact
		if err := s.loadFile(file, &artifact); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}

	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].ID < artifacts[j].ID
	})
	return artifacts, nil
}

func (s *Store) LoadEntry(id string) (model.Entry, error) {
	var entry model.Entry
	err := s.load("entries", id, &entry)
	return entry, err
}

func (s *Store) LoadEntries(ids []string) ([]model.Entry, error) {
	entries := make([]model.Entry, 0, len(ids))
	for _, id := range ids {
		entry, err := s.LoadEntry(id)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *Store) ListEntries() ([]model.Entry, error) {
	files, err := s.listObjectFiles("entries")
	if err != nil {
		return nil, err
	}

	entries := make([]model.Entry, 0, len(files))
	for _, file := range files {
		var entry model.Entry
		if err := s.loadFile(file, &entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})
	return entries, nil
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

func (s *Store) ListThreads() ([]model.Thread, error) {
	files, err := s.listObjectFiles("threads")
	if err != nil {
		return nil, err
	}

	threads := make([]model.Thread, 0, len(files))
	for _, file := range files {
		var thread model.Thread
		if err := s.loadFile(file, &thread); err != nil {
			return nil, err
		}
		threads = append(threads, thread)
	}

	sort.Slice(threads, func(i, j int) bool {
		return threads[i].ID < threads[j].ID
	})
	return threads, nil
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

func (s *Store) ListWorkspaces() ([]model.Workspace, error) {
	files, err := s.listObjectFiles("workspaces")
	if err != nil {
		return nil, err
	}

	workspaces := make([]model.Workspace, 0, len(files))
	for _, file := range files {
		var workspace model.Workspace
		if err := s.loadFile(file, &workspace); err != nil {
			return nil, err
		}
		workspaces = append(workspaces, workspace)
	}

	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].ID < workspaces[j].ID
	})
	return workspaces, nil
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

func (s *Store) ListTopics() ([]model.Topic, error) {
	files, err := s.listObjectFiles("topics")
	if err != nil {
		return nil, err
	}

	topics := make([]model.Topic, 0, len(files))
	for _, file := range files {
		var topic model.Topic
		if err := s.loadFile(file, &topic); err != nil {
			return nil, err
		}
		topics = append(topics, topic)
	}

	sort.Slice(topics, func(i, j int) bool {
		return topics[i].ID < topics[j].ID
	})
	return topics, nil
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
	return s.loadFile(s.objectPath(kind, id), target)
}

func (s *Store) objectPath(kind, id string) string {
	return filepath.Join(s.root, kind, id+".json")
}

func (s *Store) listObjectFiles(kind string) ([]string, error) {
	if !s.Initialized() {
		return nil, errors.New("store is not initialized")
	}
	pattern := filepath.Join(s.root, kind, "*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob %s: %w", kind, err)
	}
	sort.Strings(files)
	return files, nil
}

func (s *Store) loadFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
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

package model

import (
	"errors"
	"fmt"
	"time"
)

type VitalityState string

const (
	VitalityLive    VitalityState = "live"
	VitalityDormant VitalityState = "dormant"
	VitalitySettled VitalityState = "settled"
)

func (s VitalityState) Validate() error {
	switch s {
	case VitalityLive, VitalityDormant, VitalitySettled:
		return nil
	default:
		return fmt.Errorf("invalid vitality state %q", s)
	}
}

type Entry struct {
	ID             string    `json:"id"`
	Subject        string    `json:"subject"`
	State          string    `json:"state"`
	Meaning        string    `json:"meaning"`
	RelatedContext []string  `json:"related_context,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (e Entry) Validate() error {
	if e.ID == "" {
		return errors.New("entry id is required")
	}
	if e.Subject == "" {
		return errors.New("entry subject is required")
	}
	return nil
}

type ArtifactIngestMode string

const (
	ArtifactReference ArtifactIngestMode = "reference"
	ArtifactSnapshot  ArtifactIngestMode = "snapshot"
)

func (m ArtifactIngestMode) Validate() error {
	switch m {
	case ArtifactReference, ArtifactSnapshot:
		return nil
	default:
		return fmt.Errorf("invalid artifact ingest mode %q", m)
	}
}

type Artifact struct {
	ID            string             `json:"id"`
	Kind          string             `json:"kind"`
	SourceRef     string             `json:"source_ref"`
	IngestMode    ArtifactIngestMode `json:"ingest_mode"`
	ContentDigest string             `json:"content_digest,omitempty"`
	Body          string             `json:"body,omitempty"`
	Locator       *ArtifactLocator   `json:"locator,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

func (a Artifact) Validate() error {
	if a.ID == "" {
		return errors.New("artifact id is required")
	}
	if a.Kind == "" {
		return errors.New("artifact kind is required")
	}
	if a.SourceRef == "" {
		return errors.New("artifact source ref is required")
	}
	return a.IngestMode.Validate()
}

type ArtifactLocator struct {
	SearchRoots     []string      `json:"search_roots,omitempty"`
	SizeBytes       int64         `json:"size_bytes,omitempty"`
	ModTimeUnixNano int64         `json:"mod_time_unix_nano,omitempty"`
	FileIdentity    *FileIdentity `json:"file_identity,omitempty"`
}

type FileIdentity struct {
	Kind     string `json:"kind"`
	DeviceID uint64 `json:"device_id,omitempty"`
	ObjectID uint64 `json:"object_id,omitempty"`
}

type Thread struct {
	ID            string        `json:"id"`
	Question      string        `json:"question"`
	CurrentBelief string        `json:"current_belief,omitempty"`
	OpenEdge      string        `json:"open_edge,omitempty"`
	NextLean      string        `json:"next_lean,omitempty"`
	ReentryAnchor string        `json:"reentry_anchor,omitempty"`
	Vitality      VitalityState `json:"vitality"`
	WorkspaceID   string        `json:"workspace_id,omitempty"`
	SupportingIDs []string      `json:"supporting_ids,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

func (t Thread) Validate() error {
	if t.ID == "" {
		return errors.New("thread id is required")
	}
	if t.Question == "" {
		return errors.New("thread question is required")
	}
	return t.Vitality.Validate()
}

type Workspace struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	FocusThreadID string    `json:"focus_thread_id,omitempty"`
	ThreadIDs     []string  `json:"thread_ids,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (w Workspace) Validate() error {
	if w.ID == "" {
		return errors.New("workspace id is required")
	}
	if w.Name == "" {
		return errors.New("workspace name is required")
	}
	return nil
}

type Topic struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Orientation string    `json:"orientation,omitempty"`
	EntryIDs    []string  `json:"entry_ids,omitempty"`
	ThreadIDs   []string  `json:"thread_ids,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (t Topic) Validate() error {
	if t.ID == "" {
		return errors.New("topic id is required")
	}
	if t.Name == "" {
		return errors.New("topic name is required")
	}
	return nil
}

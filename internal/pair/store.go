package pair

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Record is one paired phone. Token is never stored.
type Record struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Hash string `json:"hash"`
}

// Store persists hashed device tokens.
type Store struct {
	path string
	mu   sync.Mutex
	list []Record
}

// OpenDefault loads %AppData%/Air-Link/devices.json (or equivalent).
func OpenDefault() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	dir = filepath.Join(dir, "Air-Link")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return Open(filepath.Join(dir, "devices.json"))
}

// Open a JSON file of hashed tokens.
func Open(path string) (*Store, error) {
	s := &Store{path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(data, &s.list); err != nil {
		return nil, fmt.Errorf("pair store: %w", err)
	}
	return s, nil
}

// NewToken returns a 256-bit hex device token (64 chars).
func NewToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// HashToken is SHA-256 hex of the raw token.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Store) saveLocked() error {
	data, err := json.MarshalIndent(s.list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

// Add stores the hash of token. The raw token is not written.
func (s *Store) Add(name, token string) (Record, error) {
	rec := Record{
		ID:   HashToken(token)[:16],
		Name: name,
		Hash: HashToken(token),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.list = append(s.list, rec)
	return rec, s.saveLocked()
}

// Remove deletes a paired device by ID.
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.list[:0]
	for _, r := range s.list {
		if r.ID != id {
			out = append(out, r)
		}
	}
	s.list = out
	return s.saveLocked()
}

// Rename sets the display name.
func (s *Store) Rename(id, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.list {
		if s.list[i].ID == id {
			s.list[i].Name = name
			return s.saveLocked()
		}
	}
	return fmt.Errorf("unknown device")
}

// Valid reports whether token matches a stored hash (constant time per record).
func (s *Store) Valid(token string) bool {
	want := HashToken(token)
	wb := []byte(want)
	s.mu.Lock()
	defer s.mu.Unlock()
	ok := false
	for _, r := range s.list {
		if subtle.ConstantTimeCompare([]byte(r.Hash), wb) == 1 {
			ok = true
		}
	}
	return ok
}

// List returns a copy of records (hashes included for the PC UI only).
func (s *Store) List() []Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Record, len(s.list))
	copy(out, s.list)
	return out
}

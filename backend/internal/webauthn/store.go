package webauthn

import (
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

// SessionStore holds WebAuthn session data for the duration of a ceremony.
type SessionStore struct {
	mu      sync.Mutex
	sessions map[string]*webauthn.SessionData
	ttl     time.Duration
}

// NewSessionStore creates a new in-memory session store with the given TTL.
func NewSessionStore(ttl time.Duration) *SessionStore {
	s := &SessionStore{
		sessions: make(map[string]*webauthn.SessionData),
		ttl:     ttl,
	}
	go s.cleanup()
	return s
}

func (s *SessionStore) Get(sessionID string) *webauthn.SessionData {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.sessions[sessionID]
	delete(s.sessions, sessionID)
	return data
}

func (s *SessionStore) Set(sessionID string, data *webauthn.SessionData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionID] = data
}

func (s *SessionStore) cleanup() {
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	for range tick.C {
		s.mu.Lock()
		now := time.Now()
		for id, data := range s.sessions {
			if data.Expires.Before(now) {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

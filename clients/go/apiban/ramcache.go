package apiban

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// DefaultMinUpstreamCheckInterval defines the minimum time between checks from
// the upstream apiban.org database; this is used by the caching store(s) to
// determine when to perform upstream checks
const DefaultMinUpstreamCheckInterval = 3 * time.Minute

type ramCacheStore struct {
	upstream Store

	minUpstreamCheckInterval time.Duration
	lastUpstreamTimestamp    time.Time
	lastUpstreamCheck        time.Time

	list []*Listing

	mu sync.Mutex
}

// NewRAMCacheStore returns a Store which implements a simple RAM-based cache
// of the official apiban.org database.  Pass 0 for the
// minUpstreamCheckInterval to use the default.
func NewRAMCacheStore(key string, minUpstreamCheckInterval time.Duration) (Store, error) {
	if key == "" {
		return nil, errors.New("API key is required")
	}

	if minUpstreamCheckInterval == 0 {
		minUpstreamCheckInterval = DefaultMinUpstreamCheckInterval
	}

	upstream := NewOfficialStore(key)

	// Fill the initial cache
	list, err := upstream.List()
	if err != nil {
		return nil, fmt.Errorf("failed initial cache fill: %w", err)
	}

	var lastUpstreamTimestamp time.Time
	if len(list) > 0 {
		lastUpstreamTimestamp = list[len(list)-1].Timestamp
	}

	return &ramCacheStore{
		upstream:                 upstream,
		minUpstreamCheckInterval: minUpstreamCheckInterval,
		lastUpstreamTimestamp:    lastUpstreamTimestamp,
		lastUpstreamCheck:        time.Now(),
		list:                     list,
	}, nil
}

func (r *ramCacheStore) fetch() error {
	if time.Since(r.lastUpstreamCheck) < r.minUpstreamCheckInterval {
		return nil
	}

	list, err := r.upstream.ListFromTime(r.lastUpstreamTimestamp)
	if err != nil {
		return err
	}

	for _, l := range list {
		if _, err := r.Add(l); err != nil {
			return fmt.Errorf("failed to add %s to list: %w", l.IP.String(), err)
		}
		r.lastUpstreamTimestamp = l.Timestamp
	}
	r.lastUpstreamCheck = time.Now()

	return nil
}

// Add implements Store
func (r *ramCacheStore) Add(l *Listing) (*Listing, error) {
	if l == nil || l.IP.IP.IsUnspecified() {
		return nil, errors.New("invalid IP address")
	}

	existing, err := r.Exists(l.IP.IP)
	if err != nil {
		return nil, fmt.Errorf("unexpected error checking for existing entry in list: %w", err)
	}
	if existing != nil {
		// already in list
		return existing, nil
	}

	if l.ID == "" {
		l.ID = l.IP.String()
	}
	if l.Timestamp.IsZero() {
		l.Timestamp = time.Now()
	}

	r.mu.Lock()
	r.list = append(r.list, l)
	r.mu.Unlock()

	return l, nil
}

// Exists implements Store
func (r *ramCacheStore) Exists(ip net.IP) (*Listing, error) {
	if err := r.fetch(); err != nil {
		return nil, fmt.Errorf("failed to fetch upstream data: %w", err)
	}

	for _, l := range r.list {
		if l.IP.Contains(ip) {
			// already in list
			return l, nil
		}
	}

	return nil, nil
}

// List implements Store
func (r *ramCacheStore) List() ([]*Listing, error) {
	if err := r.fetch(); err != nil {
		return nil, fmt.Errorf("failed to fetch upstream data: %w", err)
	}

	return r.list, nil
}

// ListFromTime implements Store
func (r *ramCacheStore) ListFromTime(t time.Time) ([]*Listing, error) {
	if err := r.fetch(); err != nil {
		return nil, fmt.Errorf("failed to fetch upstream data: %w", err)
	}

	for i, ip := range r.list {
		if ip.Timestamp.After(t) {
			return r.list[i:], nil
		}
	}
	return nil, nil
}

// Remove implements Store
func (r *ramCacheStore) Remove(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, l := range r.list {
		if l.ID == id {
			if i < len(r.list)-1 {
				copy(r.list[i:], r.list[i+1:])
			}
			r.list[len(r.list)-1] = nil
			r.list = r.list[:len(r.list)-1]
		}
	}

	return nil
}

// Reset empties the store
func (r *ramCacheStore) Reset() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.list = nil
	r.lastUpstreamTimestamp = defaultStartTimestamp()
	r.lastUpstreamCheck = defaultStartTimestamp()

	return nil
}

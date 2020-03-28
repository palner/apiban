/*
 * Copyright (C) 2020 Fred Posner (palner.com)
 *
 * This file is part of APIBAN.org.
 *
 * apiban-iptables-client is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or
 * (at your option) any later version
 *
 * apiban-iptables-client is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301  USA
 *
 */

package apiban

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

const (
	// RootURL is the base URI of the APIBAN.org API server
	RootURL = "https://apiban.org/api/"
)

// ErrBadRequest indicates a 400 response was received;
//
// NOTE: this is used by the server to indicate both that an IP address is not
// blocked (when calling Check) and that the list is complete (when calling
// Banned)
var ErrBadRequest = errors.New("Bad Request")

// NewOfficialStore returns a Store backed by the official apiban.org service
func NewOfficialStore(key string) Store {
	return &officialStore{
		key: key,
		hc:  http.DefaultClient,
	}
}

type officialStore struct {
	key string

	hc *http.Client
}

// Add implements Store
func (o *officialStore) Add(l *Listing) (*Listing, error) {
	panic("not implemented")
}

// Exists implements Store
func (o *officialStore) Exists(ip net.IP) (*Listing, error) {
	if o.key == "" {
		return nil, errors.New("API Key is required")
	}
	if ip == nil {
		return nil, errors.New("IP address is required")
	}

	entry, err := queryServer(o.hc, fmt.Sprintf("%s%s/check/%s", RootURL, o.key, ip.String()))
	if err == ErrBadRequest {
		// Not blocked
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, errors.New("empty entry received")
	}

	tsInt, err := strconv.ParseInt(entry.ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse entry ID %q as integer: %w", entry.ID, err)
	}

	ipNet := net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(32, 32), // v1 API always returns a unicast IPv4 address
	}

	// IP address is blocked
	return &Listing{
		ID:        ip.String(),
		Timestamp: time.Unix(tsInt, 0),
		IP:        ipNet,
	}, nil
}

// List implements Store
func (o *officialStore) List() ([]*Listing, error) {
	return o.ListFromTime(defaultStartTimestamp())
}

// ListFromTime implements Store
func (o *officialStore) ListFromTime(t time.Time) ([]*Listing, error) {
	if o.key == "" {
		return nil, errors.New("API Key is required")
	}

	var out []*Listing

	for {
		e, err := queryServer(o.hc, fmt.Sprintf("%s%s/banned/%d", RootURL, o.key, t.Unix()))
		if err != nil {
			return nil, err
		}

		if e.ID == "none" {
			// List complete
			break
		}
		if e.ID == "" {
			return nil, errors.New("empty ID received")
		}

		tsInt, err := strconv.ParseInt(e.ID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ID %q as timestamp: %w", e.ID, err)
		}
		t = time.Unix(tsInt, 0)

		for _, i := range e.IPs {
			_, ipNet, err := net.ParseCIDR(i + "/32")
			if err != nil {
				log.Printf("failed to parse %s as an IP address: %s", i, err.Error())
				continue
			}

			out = append(out, &Listing{
				ID:        ipNet.String(),
				Timestamp: t,
				IP:        *ipNet,
			})
		}
	}

	return out, nil
}

// Remove implements Store
func (o *officialStore) Remove(id string) error {
	panic("not implemented")
}

// Reset implements Store
func (o *officialStore) Reset() error {
	panic("not implemented")
}

// officialV1Entry describes a set of blocked IP addresses from APIBAN.org V1 API
type officialV1Entry struct {

	// ID is the timestamp of the next Entry
	ID string `json:"ID"`

	// IPs is the list of blocked IP addresses in this entry
	IPs []string `json:"ipaddress"`
}

// Banned returns a set of banned addresses, optionally limited to the
// specified startFrom ID.  If no startFrom is supplied, the entire current list will
// be pulled.
func Banned(key string, startFrom string) ([]*Listing, error) {
	var err error

	if key == "" {
		return nil, errors.New("API key is required")
	}

	var startFromInt int64 = 100
	if startFrom != "" {
		startFromInt, err = strconv.ParseInt(startFrom, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse startFrom timestamp %q: %w", startFrom, err)
		}
	}

	return NewOfficialStore(key).ListFromTime(time.Unix(startFromInt, 0))
}

// Check queries APIBAN.org to see if the provided IP address is blocked.
func Check(key string, ip string) (bool, error) {
	if key == "" {
		return false, errors.New("API Key is required")
	}
	if ip == "" {
		return false, errors.New("IP address is required")
	}

	l, err := NewOfficialStore(key).Exists(net.ParseIP(ip))
	if l != nil {
		return true, err
	}
	return false, err
}

func queryServer(c *http.Client, u string) (*officialV1Entry, error) {
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// StatusBadRequest (400) has a number of special cases to handle
	if resp.StatusCode == http.StatusBadRequest {
		return processBadRequest(resp)
	}
	if resp.StatusCode > 400 && resp.StatusCode < 500 {
		return nil, fmt.Errorf("client error (%d) from apiban.org: %s from %q", resp.StatusCode, resp.Status, u)
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("server error (%d) from apiban.org: %s from %q", resp.StatusCode, resp.Status, u)
	}
	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("unhandled error (%d) from apiban.org: %s from %q", resp.StatusCode, resp.Status, u)
	}

	entry := new(officialV1Entry)
	if err = json.NewDecoder(resp.Body).Decode(entry); err != nil {
		return nil, fmt.Errorf("failed to decode server response: %w", err)
	}

	return entry, nil
}

func processBadRequest(resp *http.Response) (*officialV1Entry, error) {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Read the bytes buffer into a new bytes.Reader
	r := bytes.NewReader(buf.Bytes())

	// First, try decoding as a normal entry
	e := new(officialV1Entry)
	if err := json.NewDecoder(r).Decode(e); err == nil {
		// Successfully decoded normal entry

		switch e.ID {
		case "none":
			// non-error case
		case "unauthorized":
			return nil, errors.New("unauthorized")
		default:
			// unhandled case
			return nil, ErrBadRequest
		}

		if len(e.IPs) > 0 {
			switch e.IPs[0] {
			case "no new bans":
				return e, nil
			}
		}

		// Unhandled case
		return nil, ErrBadRequest
	}

	// Next, try decoding as an errorEntry
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to re-seek to beginning of response buffer: %w", err)
	}

	type errorEntry struct {
		AddressCode string `json:"ipaddress"`
		IDCode      string `json:"ID"`
	}

	ee := new(errorEntry)
	if err := json.NewDecoder(r).Decode(ee); err != nil {
		return nil, fmt.Errorf("failed to decode Bad Request response: %w", err)
	}

	switch ee.AddressCode {
	case "rate limit exceeded":
		return nil, errors.New("rate limit exceeded")
	default:
		// unhandled case
		return nil, ErrBadRequest
	}
}

func defaultStartTimestamp() time.Time {
	return time.Now().AddDate(-1, 0, 0)
}

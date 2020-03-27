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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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

// Entry describes a set of blocked IP addresses from APIBAN.org
type Entry struct {

	// ID is the timestamp of the next Entry
	ID string `json:"ID"`

	// IPs is the list of blocked IP addresses in this entry
	IPs []string `json:"ipaddress"`
}

// Banned returns a set of banned addresses, optionally limited to the
// specified startFrom ID.  If no startFrom is supplied, the entire current list will
// be pulled.
func Banned(key string, startFrom string) (*Entry, error) {
	if key == "" {
		return nil, errors.New("API Key is required")
	}

	if startFrom == "" {
		startFrom = "100" // NOTE: arbitrary ID copied from reference source
	}

	out := &Entry{
		ID: startFrom,
	}

	for {
		e, err := queryServer(http.DefaultClient, fmt.Sprintf("%s%s/banned/%s", RootURL, key, out.ID))
		if err == ErrBadRequest {
			// End of list
			return out, nil
		}
		if err != nil {
			return nil, err
		}

		if e.ID == "none" {
			// List complete
			return out, nil
		}
		if e.ID == "" {
			return nil, errors.New("empty ID received")
		}

		// Set the next ID
		out.ID = e.ID

		// Aggregate the received IPs
		out.IPs = append(out.IPs, e.IPs...)
	}
}

// Check queries APIBAN.org to see if the provided IP address is blocked.
func Check(key string, ip string) (bool, error) {
	if key == "" {
		return false, errors.New("API Key is required")
	}
	if ip == "" {
		return false, errors.New("IP address is required")
	}

	entry, err := queryServer(http.DefaultClient, fmt.Sprintf("%s%s/check/%s", RootURL, key, ip))
	if err == ErrBadRequest {
		// Not blocked
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if entry == nil {
		return false, errors.New("empty entry received")
	}

	// IP address is blocked
	return true, nil
}

func queryServer(c *http.Client, u string) (*Entry, error) {
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Error code 400 is special for Check errors, indicating the IP address is not blacklisted, so we return a sentinel error here
	if resp.StatusCode == 400 {
		return nil, ErrBadRequest
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

	entry := new(Entry)
	if err = json.NewDecoder(resp.Body).Decode(entry); err != nil {
		return nil, fmt.Errorf("failed to decode server response: %w", err)
	}
	return entry, nil
}

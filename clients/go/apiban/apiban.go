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
	"net"
	"time"
)

// Store defines and interface for storing and retrieving entries in the APIBan database, local or remote
type Store interface {

	// Add inserts the given Listing into the store
	Add(l *Listing) (*Listing, error)

	// Exists checks to see whether the given IP matches a Listing in the store, returning the first matching Listing.
	Exists(ip net.IP) (*Listing, error)

	// List retrieves the contents of the store
	List() ([]*Listing, error)

	// ListFromTime retrieves the contents of the store from the given timestamp
	ListFromTime(t time.Time) ([]*Listing, error)

	// Remove deletes the given Listing from the store.
	Remove(id string) error

	// Reset empties the store
	Reset() error
}

// Listing is an individually-listed IP address or subnet
type Listing struct {

	// ID is the unique identifier for this Listing; for official APIBAN v1 entries, this is simply the IP address.
	ID string

	// Timestamp is the time at which this Listing was added to the apiban.org database
	Timestamp time.Time

	// IP is the IP address or IP network which is in the apiban.org database
	IP net.IPNet
}

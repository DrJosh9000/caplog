// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dashboard

// This file safely aggregates packet counts and sizes.

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"packets"
)

var (
	vals Values

	mapVars = MapValues{
		UpByIP:     make(map[string]Aggregation),
		DownByIP:   make(map[string]Aggregation),
		UpByName:   make(map[string]Aggregation),
		DownByName: make(map[string]Aggregation),
		SrcDstIP:   make(map[string]map[string]Aggregation),
		SrcDstName: make(map[string]map[string]Aggregation),
	}

	LocalNetblock *net.IPNet
	stdLocalNets  = []*net.IPNet{
		mustParseCIDR("10.0.0.0/8"), // RFC1918 IPv4 private addresses
		mustParseCIDR("172.16.0.0/12"),
		mustParseCIDR("192.168.0.0/16"),
		mustParseCIDR("fd00::/8"),           // RFC4193 IPv6 private addresses
		mustParseCIDR("169.254.0.0/16"),     // RFC3917 IPv4 link-local addresses
		mustParseCIDR("fe80::/10"),          // RFC4862 IPv6 link-local/autoconfig addresses
		mustParseCIDR("0.0.0.0/32"),         // Broadcast source
		mustParseCIDR("255.255.255.255/32"), // Broadcast destination
	}
)

func mustParseCIDR(s string) *net.IPNet {
	_, cidr, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return cidr
}

// Aggregation combines the two counters for each total or flow.
type Aggregation struct {
	Bytes, Packets uint64
}

// Add adds 1 packet of a given size to an agg, returning the new value.
func (a *Aggregation) Add(bytes uint64) {
	atomic.AddUint64(&a.Bytes, bytes)
	atomic.AddUint64(&a.Packets, 1)
}

// Values contains all the aggregations for a flow (and other values).
type Values struct {
	Now time.Time

	// Flow statistics.
	Up, Down, Internal, External, Total Aggregation
	V4, V6                              Aggregation
}

type MapValues struct {
	UpByIP, DownByIP     map[string]Aggregation
	UpByName, DownByName map[string]Aggregation
	SrcDstIP, SrcDstName map[string]map[string]Aggregation
}

// isLocal returns true if the IP is a private or link-local address. It also
// considers the LocalNetblock passed in (from a flag), useful in case NAT is
// not in use.
func isLocal(ip net.IP) bool {
	if LocalNetblock != nil && LocalNetblock.Contains(ip) {
		return true
	}
	for _, cidr := range stdLocalNets {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// AddPacket lets vals account for the packet.
func AddPacket(m *packets.Metadata) {
	vals.Total.Add(m.Size)

	// Classify packet flow for subtotals.
	srcPrivate, dstPrivate := isLocal(m.SrcIP), isLocal(m.DstIP)
	switch {
	case srcPrivate && dstPrivate:
		vals.Internal.Add(m.Size)
	case srcPrivate:
		vals.Up.Add(m.Size)
	case dstPrivate:
		vals.Down.Add(m.Size)
	default:
		vals.External.Add(m.Size)
	}

	// Only add to the V4 / V6 counters when considering internet
	// ingress/egress which is more intesting, also because monitoring
	// traffic will slowly dominate over time otherwise.
	if !(srcPrivate && dstPrivate) {
		if m.V6 {
			vals.V6.Add(m.Size)
		} else {
			vals.V4.Add(m.Size)
		}
	}
	/*
		// Per-IP accounting.
		src, dst := m.SrcIP.String(), m.DstIP.String()
		mapVars.UpByIP[src] = mapVars.UpByIP[src].Add(m.Size)
		mapVars.DownByIP[dst] = mapVars.DownByIP[dst].Add(m.Size)

		// Per-Name accounting.
		mapVars.UpByName[m.SrcName] = mapVars.UpByName[m.SrcName].Add(m.Size)
		mapVars.DownByName[m.DstName] = mapVars.DownByName[m.DstName].Add(m.Size)

		// Src-Dst by IP accounting.
		if dstMap, ok := mapVars.SrcDstIP[src]; ok {
			dstMap[dst] = dstMap[dst].Add(m.Size)
		} else {
			mapVars.SrcDstIP[src] = map[string]Aggregation{
				dst: {Bytes: m.Size, Packets: 1},
			}
		}

		// Src-Dst by name accounting.
		if dstMap, ok := mapVars.SrcDstName[m.SrcName]; ok {
			dstMap[m.DstName] = dstMap[m.DstName].Add(m.Size)
		} else {
			mapVars.SrcDstName[m.SrcName] = map[string]Aggregation{
				m.DstName: {Bytes: m.Size, Packets: 1},
			}
		}
	*/
}

// State returns the current state of the vals.
func State() Values {
	vals.Now = time.Now()
	return vals
}

func dashValuesHandler(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(State()); err != nil {
		log.Print("template failed to write:", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

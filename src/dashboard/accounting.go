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

// This file aggregates packet counts and sizes.

import (
	"encoding/json"
	"log"
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
)

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

// AddPacket lets vals account for the packet.
func AddPacket(m *packets.Metadata) {
	vals.Total.Add(m.Size)

	// Classify packet flow for subtotals.
	srcPrivate, dstPrivate := packets.IsLocal(m.SrcIP), packets.IsLocal(m.DstIP)
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

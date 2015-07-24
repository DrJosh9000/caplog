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

package packets

// This file implements a concurrent-safe reverse DNS map.

import (
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// reverseDNSMap is a concurrent-safe reverse DNS mapping (from Endpoints to names).
type reverseDNSMap struct {
	rm map[gopacket.Endpoint]string
	mu sync.RWMutex
}

// newReverseDNSMap makes an empty reverseDNSMap.
func newReverseDNSMap() *reverseDNSMap {
	return &reverseDNSMap{
		rm: make(map[gopacket.Endpoint]string),
	}
}

// name returns either the name that mapped to the given endpoint most recently,
// or the formatted endpoint if not found.
func (r *reverseDNSMap) name(e gopacket.Endpoint) string {
	if n, ok := r.rm[e]; ok {
		return n
	}
	return e.String()
}

// names maps the names for both endpoints of a flow.
func (r *reverseDNSMap) names(netFlow gopacket.Flow) (string, string) {
	src, dst := netFlow.Endpoints()
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.name(src), r.name(dst)
}

// add reads the DNS answers and adds them to the mapping.
func (r *reverseDNSMap) add(dns *layers.DNS) {
	r.mu.Lock()
	for _, a := range dns.Answers {
		// TODO: Handle CNAMEs in some fashion.
		if a.Class == layers.DNSClassIN && (a.Type == layers.DNSTypeA || a.Type == layers.DNSTypeAAAA) {
			r.rm[layers.NewIPEndpoint(a.IP)] = string(a.Name)
		}
	}
	r.mu.Unlock()
}

// len returns the number of addresses in the map.
func (r *reverseDNSMap) len() int {
	return len(r.rm)
}

// TODO: implement load/save.

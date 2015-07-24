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
	"fmt"
	"net"
	"strings"
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
	r.mu.RLock()
	defer r.mu.RUnlock()
	if n, ok := r.rm[e]; ok {
		return n
	}
	return e.String()
}

// names maps the names for both endpoints of a flow.
func (r *reverseDNSMap) names(netFlow gopacket.Flow) (string, string) {
	src, dst := netFlow.Endpoints()
	return r.name(src), r.name(dst)
}

// add reads the DNS answers and adds them to the mapping.
func (r *reverseDNSMap) add(dns *layers.DNS) {
	// Extract A, quad A, and CNAME records into useful maps.
	cnames := make(map[string]string)
	ips := make(map[gopacket.Endpoint]string)
	for _, a := range dns.Answers {
		if a.Class != layers.DNSClassIN {
			continue
		}
		switch a.Type {
		case layers.DNSTypeA, layers.DNSTypeAAAA:
			ips[layers.NewIPEndpoint(a.IP)] = string(a.Name)
		case layers.DNSTypeCNAME:
			cnames[string(a.CNAME)] = string(a.Name)
		}
	}
	// Create a topologically-sorted chain of CNAMEs resolving to each IP.
	r.mu.Lock()
	for ip, n := range ips {
		var names []string
		for ok := true; ok; n, ok = cnames[n] {
			names = append(names, n)
		}
		r.rm[ip] = strings.Join(names, ",")
	}
	r.mu.Unlock()
}

// len returns the number of addresses in the map.
func (r *reverseDNSMap) len() int {
	return len(r.rm)
}

func (r *reverseDNSMap) String() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return fmt.Sprintf("%v", r.rm)
}

// multiReverseDNS is a concurrent-safe reverse DNS mapping per host,
// so that knoweldge obtained about the DNS queries by host A doesn't
// interfere with knowledge obtained about host B.
type multiReverseDNS struct {
	maps map[gopacket.Endpoint]*reverseDNSMap
	mu   sync.RWMutex
}

// TODO: implement load/save.

func newMultiReverseDNSMap() *multiReverseDNS {
	return &multiReverseDNS{
		maps: make(map[gopacket.Endpoint]*reverseDNSMap),
	}
}

func (m *multiReverseDNS) hostMap(src gopacket.Endpoint) (rm *reverseDNSMap) {
	m.mu.RLock()
	rm = m.maps[src]
	m.mu.RUnlock()
	if rm != nil {
		return
	}
	rm = newReverseDNSMap()
	m.mu.Lock()
	m.maps[src] = rm
	m.mu.Unlock()
	return
}

func (m *multiReverseDNS) add(src net.IP, dns *layers.DNS) {
	m.hostMap(layers.NewIPEndpoint(src)).add(dns)
}

func (m *multiReverseDNS) names(src net.IP, flow gopacket.Flow) (string, string) {
	rm := m.hostMap(layers.NewIPEndpoint(src))
	return rm.names(flow)
}

// len returns the number of addresses in the map.
func (m *multiReverseDNS) len() int {
	return len(m.maps)
}

func (m *multiReverseDNS) String() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return fmt.Sprintf("%v", m.maps)
}

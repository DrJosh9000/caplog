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

import (
	"net"
	"testing"

	"github.com/google/gopacket/layers"
)

func TestSingleReverseDNSMap(t *testing.T) {
	r := newReverseDNSMap()
	ip := net.ParseIP("74.125.28.141")
	d := &layers.DNS{
		Answers: []layers.DNSResourceRecord{
			{
				Name:  []byte("golang.org"),
				Type:  layers.DNSTypeA,
				Class: layers.DNSClassIN,
				IP:    ip,
			},
		},
	}
	r.add(d)
	if got, want := "golang.org", r.name(layers.NewIPEndpoint(ip)); got != want {
		t.Errorf("name: got %q, want %q", got, want)
	}
}

func TestMultiReverseDNSMap(t *testing.T) {
	// TODO(josh): write tests
}

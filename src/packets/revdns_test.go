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
		t.Errorf("name(%v): got %q, want %q", ip, got, want)
	}
	if got, want := "1.2.3.4", r.name(layers.NewIPEndpoint(net.ParseIP("1.2.3.4"))); got != want {
		t.Errorf("name(1.2.3.4): got %q, want %q", got, want)
	}
}

func TestSingleReverseDNSMapIPv6(t *testing.T) {
	r := newReverseDNSMap()
	ip := net.ParseIP("2607:f8b0:400e:c05::8d")
	d := &layers.DNS{
		Answers: []layers.DNSResourceRecord{
			{
				Name:  []byte("golang.org"),
				Type:  layers.DNSTypeAAAA,
				Class: layers.DNSClassIN,
				IP:    ip,
			},
		},
	}
	r.add(d)
	if got, want := "golang.org", r.name(layers.NewIPEndpoint(ip)); got != want {
		t.Errorf("name(%v): got %q, want %q", ip, got, want)
	}
	if got, want := "123:456:789::abcd", r.name(layers.NewIPEndpoint(net.ParseIP("123:456:789::abcd"))); got != want {
		t.Errorf("name(123:456:789::abcd): got %q, want %q", got, want)
	}
}

func TestReverseDNSMapCNAMEChain(t *testing.T) {
	r := newReverseDNSMap()
	ip := net.ParseIP("216.58.216.14")
	d := &layers.DNS{
		Answers: []layers.DNSResourceRecord{
			{
				Name:  []byte("dl.l.google.com"),
				Type:  layers.DNSTypeA,
				Class: layers.DNSClassIN,
				IP:    ip,
			},
			{
				Name:  []byte("dl.google.com"),
				Type:  layers.DNSTypeCNAME,
				Class: layers.DNSClassIN,
				CNAME: []byte("dl.l.google.com"),
			},
		},
	}
	r.add(d)
	if got, want := "dl.l.google.com,dl.google.com", r.name(layers.NewIPEndpoint(ip)); got != want {
		t.Errorf("name(%v): got %q, want %q", ip, got, want)
	}
}

func TestMultiReverseDNSMap(t *testing.T) {
	// TODO(josh): write tests
}

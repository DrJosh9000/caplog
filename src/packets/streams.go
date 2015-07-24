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

// This file is currently pointless - but the idea was to log metadata about
// TCP streams.

import (
	"sync/atomic"

	"github.com/google/gopacket"
	"github.com/google/gopacket/tcpassembly"
)

type streamFactory struct {
	revDNS *reverseDNSMap
}

func (f *streamFactory) New(netFlow, tcpFlow gopacket.Flow) tcpassembly.Stream {
	// More accurate if reverse DNS mapping happens now.
	src, dst := f.revDNS.names(netFlow)
	return &stream{
		netFlow: netFlow,
		tcpFlow: tcpFlow,
		srcName: src,
		dstName: dst,
	}
}

type stream struct {
	netFlow, tcpFlow gopacket.Flow
	srcName, dstName string

	bytes  uint64
	closed bool
}

// Reassembled implements tcpassembly.Stream. It throws away the content
// and only accumulates the length.
func (s *stream) Reassembled(reassembly []tcpassembly.Reassembly) {
	for _, ra := range reassembly {
		atomic.AddUint64(&s.bytes, uint64(len(ra.Bytes)))
		if ra.Skip > 0 {
			atomic.AddUint64(&s.bytes, uint64(ra.Skip))
		}
	}
}

// ReassemblyComplete implements tcpassembly.Stream. It marks the stream as
// closed.
func (s *stream) ReassemblyComplete() {
	s.closed = true
}

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

// Package packets handles the core packet capturing logic (wrapping gopacket).
package packets

import (
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"

	"vars"
)

const maxBuffers = 100

// Metadata is some information about a packet, but not including the data.
type Metadata struct {
	Timestamp        time.Time
	Size             uint64
	SrcName, DstName string
	SrcIP, DstIP     net.IP
	SrcPort, DstPort uint16
	V6               bool
}

// Capture handles decoding packets and calling user functions.
type Capture struct {
	Account    func(*Metadata)
	Interface  string
	BufferSize int
	Log        func([]Metadata)

	revDNS     *reverseDNSMap
	bufferRing chan []Metadata
}

// nextBuffer returns a fresh buffer from the buffer ring, or allocates a new
// one if no buffer is ready.
func (c *Capture) nextBuffer() []Metadata {
	select {
	case b := <-c.bufferRing:
		return b
	default:
		return make([]Metadata, 0, c.BufferSize)
	}
}

// logBuffer passes the buffer to c.Log, and then tries to return the buffer
// to the buffer ring (but won't block trying).
func (c *Capture) logBuffer(b []Metadata) {
	c.Log(b)
	select {
	case c.bufferRing <- b[:0]:
	default:
	}
}

// processor is a worker that decodes packets and passes on to Account and Log.
func (c *Capture) processor(num int, packetsCh chan gopacket.Packet) {
	log.Printf("processor %d: starting", num)

	buffer := c.nextBuffer()
	defer func() {
		// TODO: Save a checkpoint.
		if c.Log != nil {
			c.Log(buffer)
		}
	}()

	var (
		eth     layers.Ethernet
		ip4     layers.IPv4
		ip6     layers.IPv6
		tcp     layers.TCP
		udp     layers.UDP
		dns     layers.DNS
		payload gopacket.Payload
	)
	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet, &eth, &ip4, &ip6, &tcp, &udp, &dns, &payload)
	for packet := range packetsCh {
		var decoded []gopacket.LayerType
		if err := parser.DecodeLayers(packet.Data(), &decoded); err != nil {
			log.Printf("processor %d: %v", num, err)
		}
		m := packet.Metadata()
		b := Metadata{
			Timestamp: m.Timestamp,
			Size:      uint64(m.Length),
		}
		for _, layerType := range decoded {
			switch layerType {
			case layers.LayerTypeIPv6:
				b.SrcIP, b.DstIP = ip6.SrcIP, ip6.DstIP
				b.SrcName, b.DstName = c.revDNS.names(ip6.NetworkFlow())
				b.V6 = true
			case layers.LayerTypeIPv4:
				b.SrcIP, b.DstIP = ip4.SrcIP, ip4.DstIP
				b.SrcName, b.DstName = c.revDNS.names(ip4.NetworkFlow())
			case layers.LayerTypeTCP:
				b.SrcPort, b.DstPort = uint16(tcp.SrcPort), uint16(tcp.DstPort)
			case layers.LayerTypeUDP:
				b.SrcPort, b.DstPort = uint16(udp.SrcPort), uint16(udp.DstPort)
			case layers.LayerTypeDNS:
				c.revDNS.add(&dns)
			}
		}

		c.Account(&b)

		if c.Log != nil {
			buffer = append(buffer, b)
			if len(buffer) >= c.BufferSize {
				go c.logBuffer(buffer)
				buffer = c.nextBuffer()
			}
		}
	}
	log.Printf("processor %d: stopping", num)
}

// Live runs a live packet capture on the interface.
func (c *Capture) Live() error {
	// Note: BlockForever != 0. 0 can do undesirable things on Darwin.
	handle, err := pcap.OpenLive(c.Interface, 1600, true, pcap.BlockForever)
	if err != nil {
		return err
	}
	defer handle.Close()
	if err := handle.SetBPFFilter("tcp or udp"); err != nil {
		return err
	}

	if c.revDNS == nil {
		c.revDNS = newReverseDNSMap()
		vars.Register("reverse-dns-map-size", vars.IntEval(c.revDNS.len).String)
	}

	packetsCh := make(chan gopacket.Packet, c.BufferSize)
	packetsChLen := func() int { return len(packetsCh) }
	vars.Register("packets-channel-len", vars.IntEval(packetsChLen).String)

	c.bufferRing = make(chan []Metadata, maxBuffers)
	bufferRingLen := func() int { return len(c.bufferRing) }
	vars.Register("buffer-ring-len", vars.IntEval(bufferRingLen).String)

	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func(num int) {
			c.processor(num, packetsCh)
			wg.Done()
		}(i)
	}

	// Pump packets into packetsCh, until interrupted.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	src := gopacket.NewPacketSource(handle, handle.LinkType())
	src.DecodeOptions = gopacket.Lazy
packetLoop:
	for {
		packet, err := src.NextPacket()
		if err == io.EOF {
			break packetLoop
		}
		if err != nil {
			log.Println("Error capturing packet:", err)
			continue
		}
		select {
		case packetsCh <- packet:
			// Nop - writing the packet to the channel was the main thing.
		case <-stop:
			log.Println("^C recieved, stopping...")
			break packetLoop
		}
	}
	// Finish processing.
	close(packetsCh)
	wg.Wait()
	return nil
}

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

// The caplog binary performs packet captures on an interface and logs the metadata - protocol,
// source and destination IP, port numbers, packet size - to an InfluxDB.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"

	"dashboard"
	"packets"
	"vars"
)

var (
	bufferSize = flag.Int("buffer", 10000, "Buffer size.")

	interfaceName = flag.String("if", "br0", "Interface to perform capture on.")
	influxDB      = flag.String("influx", "", "Destination InfluxDB for packet data.")

	port = flag.Int("port", 8080, "Serving port for user interface.")

	localNetblock = flag.String("localnet", "", "Additional netblock of routable addresses to consider local (fd::/8, 10/8, 192.168/16, etc are all automatically local).")
)

type influxEndpoint string

// jsonArray formats a Metadata point as a JSON array of values.
// This is a convenient format for Influx.
func jsonArray(w io.Writer, p *packets.Metadata) error {
	_, err := fmt.Fprintf(w, `[%d, "%v", "%v", %d, %d, "%s", "%s", %d]`,
		p.Timestamp.UnixNano()/1e6, p.SrcIP, p.DstIP, p.SrcPort, p.DstPort, p.SrcName, p.DstName, p.Size,
	)
	return err
}

// writeToInflux writes an entire buffer to the InfluxDB.
func (e influxEndpoint) writePackets(data []packets.Metadata) {
	if len(data) == 0 {
		return
	}
	log.Printf("Writing %d points to Influx...", len(data))
	pr, pw := io.Pipe()
	go func() {
		pw.Write([]byte(`[{"name":"packet","columns":["time","src_ip","dst_ip","src_port","dst_port","src_name","dst_name","size"], "points" : [`))
		first := true
		for _, p := range data {
			if first {
				first = false
			} else {
				pw.Write([]byte(","))
			}
			jsonArray(pw, &p)
		}
		pw.Write([]byte(`]}]`))
		pw.Close()
	}()
	//log.Printf("Writing %q\n", b.String())
	resp, err := http.Post(string(e), "application/json", pr)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(resp.Status)
}

func main() {
	flag.Parse()

	// For now, crank up the MAXPROCS. Something to not worry about in future versions of Go, which will use ~NumCPU maxprocs by default.
	numCPU := runtime.NumCPU()
	log.Printf("GOMAXPROCS %d -> %d\n", runtime.GOMAXPROCS(numCPU), numCPU)

	if localNetblock != nil && *localNetblock != "" {
		_, cidr, err := net.ParseCIDR(*localNetblock)
		if err != nil {
			fmt.Fprintf(os.Stderr, "-localnet must be a valid netblock: %v\n", err)
		}
		dashboard.LocalNetblock = cidr
	}

	// Serve HTTP UI.
	dashboard.RegisterHandlers()
	vars.RegisterHandler()
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
			log.Print("ListenAndServe: ", err)
		}
	}()

	c := &packets.Capture{
		Account:    dashboard.AddPacket,
		Interface:  *interfaceName,
		BufferSize: *bufferSize,
	}

	if influxDB != nil && *influxDB != "" {
		epURL, err := url.Parse(*influxDB)
		if err != nil {
			panic(err)
		}
		epURL.Path = "db/caplog/series"
		// TODO: Put the InfluxDB user/password somewhere better.
		epURL.RawQuery = url.Values{
			"u": []string{"caplog"},
			"p": []string{"freshbeans"},
		}.Encode()
		endpoint := influxEndpoint(epURL.String())
		c.Log = endpoint.writePackets
	}

	if err := c.Live(); err != nil {
		panic(err)
	}
}

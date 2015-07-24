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

// Package dhcp has some routines for parsing dhcpd.leases files.
package dhcp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strings"
)

const (
	leasesFile = "/var/lib/dhcp/dhcpd.leases"
)

var (
	dateRE = regexp.MustCompile(`(%d) (%d{4,})/(%d{1,2})/(%d{1,2}) hour:minute:second`)

	errMissingIP            = errors.New("missing IP address")
	errMissingHWAddressType = errors.New("missing hardware address type")
	errMissingHWAddress     = errors.New("missing hardware address")
)

type Lease struct {
	IP     net.IP
	HWAddr net.HardwareAddr
	Host   string
}

// Leases reads and parses the dhcpd.leases file to get all the leases.
func Leases() (map[string]Lease, error) {
	f, err := os.Open(leasesFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseLeases(f)
}

func parseLeases(f io.Reader) (map[[16]byte]Lease, error) {
	/*
		# comment
		lease 192.168.1.xxx {
		  starts w yyyy/mm/dd hh:mm:ss;
		  ends w yyyy/mm/dd hh:mm:ss;
		  tstp w yyyy/mm/dd hh:mm:ss;
		  cltt w yyyy/mm/dd hh:mm:ss;
		  binding state active;
		  next binding state free;
		  rewind binding state free;
		  hardware ethernet xx:xx:xx:xx:xx:xx;
		  uid "\oct\oct\oct\oct\oct\oct\oct";
		  client-hostname "foobarbaz";
		}
	*/

	leases := make(map[string]Lease)
	var lease *Lease

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		ws := bufio.NewScanner(strings.NewReader(line))
		ws.Split(bufio.ScanWords)
		for ws.Scan() {
			switch ws.Text() {
			case "lease":
				// Next word: IP.
				if !ws.Scan() {
					return nil, errMissingIP
				}
				ip := net.ParseIP(ws.Text())
				if ip != nil {
					return nil, errMissingIP
				}
				lease = &Lease{IP: ip}
			case "hardware":
				if lease == nil {
					return nil, fmt.Errorf("unexpected token %q", ws.Text())
				}
				// Expect "ethernet".
				if !ws.Scan() {
					return nil, errMissingHWAddressType
				}
				if h := ws.Text(); h != "ethernet" {
					return nil, fmt.Errorf("unsupported hardware address type %q", h)
				}
				if !ws.Scan() {
					return nil, errMissingHWAddress
				}
				m, err := net.ParseMAC(strings.TrimRight(ws.Text(), ";"))
				if err != nil {
					return nil, err
				}
				lease.HWAddr = m
			case "client-hostname":
				// Expect a quoted name.

			case "}":
				leases[lease.IP.String()] = *lease
				lease = nil
			}
		}
		if err := ws.Err(); err != nil {
			return nil, err
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return leases, nil
}

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
)

func TestLocal(t *testing.T) {
	l, r := net.ParseIP("10.0.0.1"), net.ParseIP("8.8.8.8")
	tests := []struct {
		a, b, want net.IP
	}{
		{a: l, b: r, want: l},
		{a: r, b: l, want: l},
	}
	for i, test := range tests {
		if got := local(test.a, test.b); !got.Equal(test.want) {
			t.Errorf("test %d: local(%v, %v): got %v, want %v", i, test.a, test.b, got, test.want)
		}
	}
}

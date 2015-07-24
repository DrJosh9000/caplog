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

// Package vars exposes some of the app's statistics via the HTTP server.
package vars

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
)

var varMap = map[string]VarEval{
	"go-version":    runtime.Version,
	"go-root":       runtime.GOROOT,
	"num-cpu":       IntEval(runtime.NumCPU).String,
	"num-cgo-call":  Int64Eval(runtime.NumCgoCall).String,
	"num-goroutine": IntEval(runtime.NumGoroutine).String,
}

type VarEval func() string

type IntEval func() int

func (i IntEval) String() string {
	return fmt.Sprintf("%d", i())
}

type Int64Eval func() int64

func (i Int64Eval) String() string {
	return fmt.Sprintf("%d", i())
}

type Uint64Eval func() uint64

func (i Uint64Eval) String() string {
	return fmt.Sprintf("%d", i())
}

// Register registers a var handler (produces a formatted value for a key).
func Register(key string, eval VarEval) {
	varMap[key] = eval
}

// Uint64 registers a handler that just prints the current value of an uint64.
// Be careful that your integer doesn't move around!
func Uint64(key string, i *uint64) {
	Register(key, func() string {
		return fmt.Sprintf("%d", *i)
	})
}

// String registers a handler that prints the current value of a string.
func String(key string, s *string) {
	Register(key, func() string {
		return *s
	})
}

// Evaluate evaluates every var and organises the values into a map.
func Evaluate() map[string]string {
	m := make(map[string]string, len(varMap))
	for k, ev := range varMap {
		m[k] = ev()
	}
	return m
}

func handler(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(Evaluate()); err != nil {
		log.Print("template failed to write:", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// RegisterHandler adds a HTTP handler for the vars endpoint.
func RegisterHandler() {
	http.HandleFunc("/vars", handler)
}

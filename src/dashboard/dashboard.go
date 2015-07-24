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

// Package dashboard exposes a pretty dashboard of collected data for human consumption
// via the HTTP server.
package dashboard

import (
	"html/template"
	"log"
	"net/http"
)

const (
	// TODO: Have a templates dir installed somewhere sensible.
	dashTemplateBase    = "/home/josh/caplog/src/dashboard/"
	dashTemplateFile    = dashTemplateBase + "dashboard.html"
	ipTableTemplateFile = dashTemplateBase + "srcdsttable.html"
)

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Load the template each call; because makes dev easier.
	// TODO: Move template parsing back out, make template static.
	dash, err := template.ParseFiles(dashTemplateFile, ipTableTemplateFile)
	if err != nil {
		log.Print("template failed to parse:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := dash.ExecuteTemplate(w, "dashboard.html", State()); err != nil {
		log.Print("template failed to write:", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func RegisterHandlers() {
	http.HandleFunc("/dashboard/json", dashValuesHandler)
	http.HandleFunc("/dashboard", dashboardHandler)
}

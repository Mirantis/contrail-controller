/*
Copyright 2016 Juniper Networks, Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package agent

import (
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"
)

const (
	xmlSample = "vrouter_itf_req.xml"
)

func TestParseLinkLocalIpFromBuf(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	directory := filepath.Dir(file)
	xmlPath := filepath.Join(directory, xmlSample)
	buf, err := ioutil.ReadFile(xmlPath)
	if err != nil {
		t.Fatal(err)
	}
	linkLocalIP, err := parseLinkLocalIPFromBuf(buf)
	if err != nil {
		t.Fatal(err)
	}
	if expectedIP := "169.254.0.4"; linkLocalIP != expectedIP {
		t.Fatalf("Expected IP %v is not equal to parsed %v", linkLocalIP, expectedIP)
	}
}

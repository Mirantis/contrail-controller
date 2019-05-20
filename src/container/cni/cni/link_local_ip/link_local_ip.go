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
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const (
	vrouterItfReqUrl = "http://127.0.0.1:8085/Snh_ItfReq?x="
	linkLocalIpAttr  = "mdata_ip_addr"
)

type ItfRespList struct {
	ItfResp    interface{}
	Pagination interface{}
}

// GetLinkLocalIP accepts system interface name and returns link local ip
// fetched from vrouter
func GetLinkLocalIP(systemItf string) (string, error) {
	client := http.Client{
		Timeout: 10 * time.Second,
	}
	response, err := client.Get(strings.Join([]string{vrouterItfReqUrl, systemItf}, ""))
	if err != nil {
		return "", fmt.Errorf("wasnt able to receive information on interface %s from %s: %v",
			systemItf, vrouterItfReqUrl, err)
	}
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}
	linkLocal, err := parseLinkLocalIPFromBuf(buf)
	if err != nil {
		return "", fmt.Errorf("error parsing link local: %v", err)
	}
	if linkLocal == "0.0.0.0" {
		return "", fmt.Errorf("link local ip wasnt yet assigned")
	}
	return linkLocal, nil
}

func parseLinkLocalIPFromBuf(buf []byte) (string, error) {
	tokens, err := collectMdataIPAddrTokens(buf)
	if err != nil {
		return "", fmt.Errorf("error parsing mdata_ip_addr_tokens: %v", err)
	}
	if len(tokens) == 0 {
		return "", fmt.Errorf("no single token with link local ip was returned")
	}
	if len(tokens) > 1 {
		return "", fmt.Errorf("more than one token was returned")
	}
	return tokens[0], nil
}

func collectMdataIPAddrTokens(buf []byte) ([]string, error) {
	decoder := xml.NewDecoder(bytes.NewBuffer(buf))
	tokens := []string{}
	for {
		t, err := decoder.Token()
		if t == nil {
			break
		}
		if err != nil {
			return tokens, err
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == linkLocalIpAttr {
				var token string
				err = decoder.DecodeElement(&token, &se)
				if err != nil {
					return tokens, fmt.Errorf("error parsing token %v: %v", se, err)
				}
				tokens = append(tokens, token)
			}
		}
	}
	return tokens, nil
}

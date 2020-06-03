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

package iptables

import (
	"flag"
	"reflect"
	"testing"
)

var SUDO bool

func init() {
	flag.BoolVar(&SUDO, "sudo", false, "sudo flag allows to run potentially desctructive tests that are relying on host OS")
}

func skipIfNotSudo(t *testing.T) {
	if !SUDO {
		t.Skip("test relies on sudo flag")
	}
}

func TestMatcherForCommentedRules(t *testing.T) {
	output := `-P OUTPUT ACCEPT
-A OUTPUT -d 10.150.255.249/32 -m comment --comment "kube-system:kube-dns-692631775-3r9ck" -j DNAT --to-destination 169.254.0.5
-A OUTPUT ! -d 127.0.0.0/8 -m addrtype --dst-type LOCAL -j DOCKER`
	rules := matchRulesByComment(output, "kube-system:kube-dns-692631775-3r9ck")
	if lth := len(rules); lth < 1 {
		t.Fatalf("Length of rules %+v expected to be 1, but got %d", rules, lth)
	}
	expectedRule := []string{"-A", "OUTPUT", "-d", "10.150.255.249/32", "-m", "comment",
		"--comment", "kube-system:kube-dns-692631775-3r9ck", "-j", "DNAT", "--to-destination", "169.254.0.5"}
	if !reflect.DeepEqual(rules[0], expectedRule) {
		t.Fatalf("Expected to match different rule. Got %+v. Expected %+v", rules[0], expectedRule)
	}
}

func TestCreateTestChain(t *testing.T) {
	skipIfNotSudo(t)
	testCases := []struct {
		chain           string
		expectedCreated bool
		err             bool
	}{
		{"TEST", true, false},
		{"TEST", false, false},
		{"ANOTHER-TEST", true, false},
	}
	chainsToClean := []string{}
	for i, tc := range testCases {
		t.Run(tc.chain, func(t *testing.T) {
			t.Logf("%d: test for chain %s - expect %t", i, tc.chain, tc.expectedCreated)
			created, err := createChain(tc.chain, natTable)
			if err != nil && !tc.err {
				t.Errorf("unexpected error %v", err)
			}
			if tc.expectedCreated != created {
				t.Errorf("unexpected result for creation of chain %s", tc.chain)
			}
			if created {
				chainsToClean = append(chainsToClean, tc.chain)
			}
		})
	}
	t.Logf("Removing chains %v", chainsToClean)
	for _, chain := range chainsToClean {
		if err := deleteChain(chain, natTable); err != nil {
			t.Errorf("error deleting chain %s: %v", chain, natTable)
		}
	}
}

func testError(t *testing.T, f func() error) {
	if err := f(); err != nil {
		t.Error(err)
	}
}

func TestAddDeleteDnatRules(t *testing.T) {
	skipIfNotSudo(t)
	testError(t, EnableContrailChains)
	testError(t, func() error {
		return AddDnatRuleFromToWithComment("10.170.0.1", "10.180.0.1", "test-comment")
	})
	testError(t, func() error {
		return DeleteDnatRulesByComment("test-comment")
	})
	testError(t, DeleteContrailChains)
}

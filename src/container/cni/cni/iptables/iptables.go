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
	"bytes"
	"fmt"
	log "../logging"
	"os/exec"
	"strings"
)

const (
	commentField            = "--comment"
	chainExistError         = "iptables: Chain already exists."
	contrailDnatChain       = "CONTRAIL-CNI-LOCAL-DNAT"
	contrailMasqueradeChain = "CONTRAIL-CNI-MASQUERADE"
	contrailIface           = "vhost0"
	natTable                = "nat"
	iptablesBin             = "iptables"
)

// RunCommand executes specified command logs info and errors
func RunCommand(name string, cmdParts ...string) (string, string, error) {
	cmdParts = append(cmdParts, "-w 5")
	log.Infof("Running command: %s %+v\n", name, cmdParts)
	cmd := exec.Command(name, cmdParts...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	if err != nil {
		msg := fmt.Sprintf("Command %+v crashed. stdout: %s. stderr: %s . err %v",
			cmdParts, string(stdout), stderr.String(), err)
		log.Info(msg)
		return "", stderr.String(), fmt.Errorf(msg)
	}
	log.Infof("Stdout for command %+v - %s", cmdParts, stdout)
	return string(stdout), stderr.String(), nil
}

// CreateChain returns true only if chain was created, otherwise false
func createChain(chain, table string) (bool, error) {
	_, stderr, err := RunCommand(iptablesBin, "-N", chain, "-t", table)
	if err == nil {
		return true, nil
	}
	if strings.TrimSpace(stderr) == chainExistError {
		log.Infof("chain %s already exists", chain)
		return false, nil
	}
	return false, err
}

func deleteChain(chain, table string) error {
	_, stderr, err := RunCommand(iptablesBin, "-F", chain, "-t", table)
	if err != nil {
		return fmt.Errorf("error flushing chain %s, %s: %v", chain, stderr, err)
	}
	_, stderr, err = RunCommand(iptablesBin, "-X", chain, "-t", table)
	if err != nil {
		return fmt.Errorf("error deleting chain %s, %s: %v", chain, stderr, err)
	}
	return err
}

func unlinkChain(parentChain, targetChain, table string) error {
	_, _, err := RunCommand(iptablesBin, "-D", parentChain, "-j", targetChain, "-t", table)
	if err != nil {
		return fmt.Errorf("error deleting link to parent %s from target %s: %v", parentChain, targetChain, err)
	}
	return nil
}

// EnableContrailChains creates known contrail chains and adds them to appropriate default chains
func EnableContrailChains() error {
	var errMasq error
	addMasq := func() {
		errMasq = AddMasqueradeRuleForIface(contrailIface)
	}
	if err := EnableChainAsTarget("OUTPUT", contrailDnatChain, natTable, func() {}); err != nil {
		return err
	}
	if err := EnableChainAsTarget("POSTROUTING", contrailMasqueradeChain, natTable, addMasq); err != nil {
		return err
	}
	if errMasq != nil {
		return fmt.Errorf("error adding masq rule: %v", errMasq)
	}
	return nil
}

// DeleteContrailChains will flush and delete known contrail cni chains
func DeleteContrailChains() error {
	if err := unlinkChain("OUTPUT", contrailDnatChain, natTable); err != nil {
		return err
	}
	if err := deleteChain(contrailDnatChain, natTable); err != nil {
		return fmt.Errorf("error deleteing %s: %v", contrailDnatChain, err)
	}
	if err := unlinkChain("POSTROUTING", contrailMasqueradeChain, natTable); err != nil {
		return err
	}
	if err := deleteChain(contrailMasqueradeChain, natTable); err != nil {
		return fmt.Errorf("error deleteing %s: %v", contrailMasqueradeChain, err)
	}
	return nil
}

// EnableChainAsTarget add chain as target if it was created
func EnableChainAsTarget(parentChain, targetChain, table string, onCreation func()) error {
	created, err := createChain(targetChain, table)
	if err != nil {
		return fmt.Errorf("error creating chain %s: %v", targetChain, err)
	}
	if created {
		_, _, err := RunCommand(iptablesBin, "-I", parentChain, "-j", targetChain, "-t", table)
		if err != nil {
			return fmt.Errorf("error adding chain %s as target for %s: %v", targetChain, parentChain, err)
		}
		onCreation()
	}
	return nil
}

func masqueradeTemplate(method, iface string) []string {
	return []string{method, contrailMasqueradeChain, "-t", natTable, "-o", iface, "-j", "MASQUERADE"}
}

func dnatForOutputTemplate(method, from, to, comment string) []string {
	return []string{method, contrailDnatChain, "-t", natTable, "-j", "DNAT", "-d", from,
		"--to-destination", to, "-m", "comment", "--comment", comment}
}

// iptables -t nat -I CONTRAIL-CNI-MASQUERADE -o vhost0 -j MASQUERADE
func AddMasqueradeRuleForIface(iface string) error {
	_, _, err := RunCommand(iptablesBin, masqueradeTemplate("-C", iface)...)
	if err != nil {
		_, _, err = RunCommand(iptablesBin, masqueradeTemplate("-I", iface)...)
	}
	return err
}

// iptables -t nat -I OUTPUT -j CONTRAIL-CNI-DNAT -d 10.150.255.244 --to-destination 169.254.0.4 --comment "default:nginx"
func AddDnatRuleFromToWithComment(from, to, comment string) error {
	_, _, err := RunCommand(iptablesBin, dnatForOutputTemplate("-C", from, to, comment)...)
	if err != nil {
		_, _, err = RunCommand(iptablesBin, dnatForOutputTemplate("-I", from, to, comment)...)
	}
	return err
}

// DeleteDnatRulesByComment matches rules from OTPUTs chain nat table and removes all
// that were found.
func DeleteDnatRulesByComment(comment string) error {
	cmd := []string{"-S", contrailDnatChain, "-t", natTable}
	output, _, err := RunCommand(iptablesBin, cmd...)
	if err != nil {
		return fmt.Errorf("error running iptables %+v: %v", cmd, err)
	}
	for _, rule := range matchRulesByComment(string(output), comment) {
		if err := deleteRule(rule); err != nil {
			return err
		}
	}
	return nil
}

func deleteRule(rule []string) error {
	rule[0] = "-D"
	extendedRule := make([]string, 0, len(rule)+2)
	extendedRule = append(extendedRule, "-t", natTable)
	extendedRule = append(extendedRule, rule...)
	_, _, err := RunCommand(iptablesBin, extendedRule...)
	if err != nil {
		return fmt.Errorf("deleting rule failed %+v: %v", extendedRule, err)
	}
	return nil
}

func addBracketsToComment(comment string) string {
	return strings.Join([]string{"\"", comment, "\""}, "")
}

func matchRulesByComment(output, comment string) [][]string {
	lines := strings.Split(output, "\n")
	// in majority of cases we will have only one rule matched, thus just
	// prepare memory for this one rule
	result := make([][]string, 0, 1)
	// see a test for explanation
	comment = addBracketsToComment(comment)
	for _, line := range lines {
		cmd := strings.Fields(line)
		commentFieldIndex := -1
		for i, field := range cmd {
			if field == commentField {
				commentFieldIndex = i
			}
		}
		if commentFieldIndex >= 0 && cmd[commentFieldIndex+1] == comment {
			commentFull := cmd[commentFieldIndex+1]
			cmd[commentFieldIndex+1] = commentFull[1 : len(commentFull)-1]
			result = append(result, cmd)
		}
	}
	return result
}

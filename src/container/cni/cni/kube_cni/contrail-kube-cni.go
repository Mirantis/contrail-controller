// vim: tabstop=4 expandtab shiftwidth=4 softtabstop=4
//
// Copyright (c) 2017 Juniper Networks, Inc. All rights reserved.
//
/****************************************************************************
 * Main routines for kubernetes CNI plugin
 ****************************************************************************/
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"../contrail"
	log "../logging"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func BytesToStrings(data []byte) []string {
	var out []string
	for _, part := range bytes.Split(data, []byte{0}) {
		out = append(out, string(part))
	}
	return out
}

func getKubeConfigPath() (string, error) {

	out, err := exec.Command("pgrep", "-f", "kubelet").Output()
	if err != nil {
		return "", err
	}
	processID := strings.TrimSpace(string(out))

	cmdline, err := ioutil.ReadFile(fmt.Sprintf("/proc/%s/cmdline", processID))
	if err != nil {
		return "", err
	}
	args := BytesToStrings(cmdline)
	for _, arg := range args {
		if strings.HasPrefix(arg, "--kubeconfig=") {
			return arg[13:], nil
		}
	}
	return "", nil
}

func getPodUidAndNodeNameFromK8sAPI(podName string, podNs string) (string, string, error) {

	kubeconfig, err := getKubeConfigPath()
	if err != nil {
		log.Errorf("ERROR in getting kubeconfig path: %s", err)
		return "", "", err
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Errorf("BuildConfigFromFlags: %s", err)
		return "", "", err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("NewForConfig: %s", err)
		return "", "", err
	}

	pod, err := clientset.CoreV1().Pods(podNs).Get(podName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("clientset.CoreV1.Pods.Get for %s: %s", podName, err)
		return "", "", err
	}

	return string(pod.UID), pod.Spec.NodeName, nil
}

// Use K8s client to get POD_container uuid and nodename from K8s API
func getPodInfo(skelArgs *skel.CmdArgs) (string, string, error) {
	var podName, podNs string
	args := strings.Split(skelArgs.Args, ";")
	for _, arg := range args {
		atr := strings.Split(arg, "=")
		if atr[0] == "K8S_POD_NAME" {
			podName = atr[1]
		}
		if atr[0] == "K8S_POD_NAMESPACE" {
			podNs = atr[1]
		}

	}
	if len(podName) == 0 {
		return "", "", errors.New("Cannot get POD name from Args")
	}
	if len(podNs) == 0 {
		return "", "", errors.New("Cannot get POD namespace from Args")
	}

	return getPodUidAndNodeNameFromK8sAPI(podName, podNs)
}

// Add command
func CmdAdd(skelArgs *skel.CmdArgs) error {
	// Initialize ContrailCni module
	cni, err := contrailCni.Init(skelArgs)
	if err != nil {
		return err
	}

	log.Infof("Came in Add for container %s", skelArgs.ContainerID)
	// Get UUID and Name for container
	containerUuid, containerName, err := getPodInfo(skelArgs)
	if err != nil {
		log.Errorf("Error getting UUID/Name for Container")
		return err
	}

	// Update UUID and Name for container
	cni.Update(containerName, containerUuid, "")
	cni.Log()

	// Handle Add command
	err = cni.CmdAdd()
	if err != nil {
		log.Errorf("Failed processing Add command.")
		return err
	}

	return nil
}

// Del command
func CmdDel(skelArgs *skel.CmdArgs) error {
	// Initialize ContrailCni module
	cni, err := contrailCni.Init(skelArgs)
	if err != nil {
		return err
	}

	log.Infof("Came in Del for container %s", skelArgs.ContainerID)
	// Get UUID and Name for container
	containerUuid, containerName, err := getPodInfo(skelArgs)
	if err != nil {
		log.Errorf("Error getting UUID/Name for Container")
		return err
	}

	// Update UUID and Name for container
	cni.Update(containerName, containerUuid, "")
	cni.Log()

	// Handle Del command
	err = cni.CmdDel()
	if err != nil {
		log.Errorf("Failed processing Del command.")
		return err
	}

	return nil
}

func main() {
	// Let CNI skeletal code handle demux based on env variables
	skel.PluginMain(CmdAdd, CmdDel,
		version.PluginSupports(contrailCni.CniVersion))
}

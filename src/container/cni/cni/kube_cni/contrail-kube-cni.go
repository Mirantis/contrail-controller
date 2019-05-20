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

func getPodUidAndNodeNameFromK8sAPI(podName string, podNs string, kubeconfig string) (string, string, error) {

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Errorf("BuildConfigFromFlags: %s", err)
		return "", "", err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("Error creating docker client. %+v", err)
		return "", "", err
	}

	pod, err := clientset.CoreV1().Pods(podNs).Get(podName, metav1.GetOptions{})
	if err != nil {
		// if pod does not exsist in K8s it shouldn't exists at all
		log.Errorf("clientset.CoreV1.Pods.Get for pod %s: NS: %s Error: %s", podName, podNs, err)
		return "", "", fmt.Errorf("Cannot get Pod.UID from K8s API")
	}
	return string(pod.UID), podName, nil
}

// Use K8s client to get POD_container uuid and nodename from K8s API
func getPodInfo(skelArgs *skel.CmdArgs, oper string, kubeconfig string) (string, string, error) {
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
	if oper == "DEL" {
		return "", podName, nil
	}
	containerUuid, containerName, err := getPodUidAndNodeNameFromK8sAPI(podName, podNs, kubeconfig)
	log.Infof("From K8s received:  containerUuid: %s containerName: %s ", containerUuid, containerName)
	return containerUuid, containerName, err
}

// Add command
func CmdAdd(skelArgs *skel.CmdArgs) error {
	// Initialize ContrailCni module
	cni, k8s, err := contrailCni.Init(skelArgs)
	if err != nil {
		return err
	}

	log.Infof("Came in Add for container %s", skelArgs.ContainerID)
	// Get UUID and Name for container
	containerUuid, containerName, err := getPodInfo(skelArgs, "ADD", k8s.Kubeconfig)
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
	cni, k8s, err := contrailCni.Init(skelArgs)
	if err != nil {
		log.Errorf("Cannot initialize ContrailCNI for %s", skelArgs)
		return err
	}

	log.Infof("Came in Del for container %s", skelArgs.ContainerID)
	// Get UUID and Name for container
	containerName, _, err := getPodInfo(skelArgs, "DEL", k8s.Kubeconfig)
	if err != nil {
		log.Errorf("Error getting UUID/Name for Container")
		return err
	}

	// Update UUID and Name for container
	cni.Update(containerName, "", "")
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

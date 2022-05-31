package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	input = `nodes=kind-worker2,kind-control-plane
value="reboot"
period="once"
scheduled="no"
timeFrame=""`
)

type State struct {
	Nodes     []string
	Value     string
	Period    string
	Scheduled string
	TimeFrame string
}

func convert(mapData string) State {
	state := &State{}
	for _, row := range strings.Split(input, "\n") {
		splittedRow := strings.Split(row, "=")
		if len(splittedRow) != 2 {
			continue // if the input is not properly formatted - do nothing
		} else {
			switch splittedRow[0] {
			case "nodes":
				state.Nodes = append(state.Nodes, strings.Split(splittedRow[1], ",")...)
				//for _, node := range state.Nodes {
				//fmt.Println(node)
				//if nodeID == node {
				//	fmt.Println("AHOY!!!")
				//}
				//}
			case "value":
				state.Value = splittedRow[1]
			case "period":
				state.Period = splittedRow[1]
			case "scheduled":
				state.Scheduled = splittedRow[1]
			case "timeFrame":
				state.TimeFrame = splittedRow[1]
			}
		}
	}
	//fmt.Printf("HERE IS THE RESULT: %+v\n", state)
	return *state
}

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", "/home/user/.kube/config")
	if err != nil {
		log.Fatal(err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	configMaps, err := client.CoreV1().ConfigMaps("kube-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Error: %v\n", err)
	}

	var state State
	for _, configMap := range configMaps.Items {
		if configMap.Name == "kured-brain" {
			state = convert(configMap.Data["state2"])
		}
	}
	fmt.Printf("%+v\n", state)
}

package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"

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

const (
	stateStrTemplate = `nodes={{.Nodes}}
value={{.Value}}
period={{.Period}}
scheduled={{.Scheduled}}
timeFrame={{.TimeFrame}}
nodeInProcess={{.NodeInProcess}}`
)

type State struct {
	Nodes         []string
	Value         string
	Period        string
	Scheduled     string
	TimeFrame     string
	NodeInProcess string
}

// parsing configMap values
func parseConfigMapVals(mapData string) State {
	state := &State{}
	for _, row := range strings.Split(mapData, "\n") {
		splittedRow := strings.Split(row, "=")
		if len(splittedRow) != 2 {
			continue // if the input is not properly formatted - do nothing
		} else {
			switch splittedRow[0] {
			case "nodes":
				state.Nodes = append(state.Nodes, strings.Split(splittedRow[1], ",")...)
			case "value":
				state.Value = splittedRow[1]
			case "period":
				state.Period = splittedRow[1]
			case "scheduled":
				state.Scheduled = splittedRow[1]
			case "timeFrame":
				state.TimeFrame = splittedRow[1]
			case "nodeInProcess":
				state.NodeInProcess = splittedRow[1]
			}
		}
	}
	//fmt.Printf("HERE IS THE RESULT: %+v\n", state)
	return *state
}

func PackConfigMapVals(state State) (stringState string) {
	var tpl bytes.Buffer
	tmpl, err := template.New("stateTempl").Parse(stateStrTemplate)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(&tpl, state)
	if err != nil {
		panic(err)
	}
	stringState = tpl.String()
	return
}

func stripBracketsFromString(state string) (processedState string) {
	for index, row := range strings.Split(state, "\n") {
		if strings.Contains(row, "nodes=[") {
			replacer := strings.NewReplacer("[", "", "]", "", " ", ",")
			row = replacer.Replace(row)
		}
		if index < (len(strings.Split(state, "\n")) - 1) {
			processedState += row + "\n"
		} else {
			processedState += row
		}
	}
	return
}

func setConfigMapKey(kubeconfPath, configMapName, namespace, stateKey, stringState string) (err error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfPath)
	if err != nil {
		return
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return
	}

	ctx := context.Background()
	configMaps, err := client.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}
	for _, configMap := range configMaps.Items {
		if configMap.GetName() == configMapName {
			configMap.Data[stateKey] = stringState
			_, err = client.CoreV1().ConfigMaps(namespace).Update(ctx, &configMap, metav1.UpdateOptions{})
			if err != nil {
				return
			}
		}
	}
	return
}

func returnConfigMapKey(kubeconfPath, key, configMapName, namespace string) (state State, err error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfPath)
	if err != nil {
		return
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return
	}

	ctx := context.Background()
	configMaps, err := client.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}

	for _, configMap := range configMaps.Items {
		if configMap.GetName() == configMapName {
			state = parseConfigMapVals(configMap.Data[key])
		}
	}
	return
}

func rebootRequired(nodeID, kubeconfPath, configMapName, namespace, stateKey string) (error, bool) {
	for i := 1; i < 5; i++ {
		if os.Getenv(fmt.Sprintf("STATE%v", i)) != "" {
			stateA := parseConfigMapVals(os.Getenv(fmt.Sprintf("STATE%v", i)))
			log.Printf("HERE: %+v\n", stateA)
			if stateA.Value == "reboot" && stateA.NodeInProcess == "" {
				stateA.NodeInProcess = nodeID
				err := setConfigMapKey(kubeconfPath, configMapName, namespace, stateKey, stripBracketsFromString(PackConfigMapVals(stateA)))
				if err != nil {
					return nil, false
				}
				return nil, true
			} else {
				if stateA.NodeInProcess == nodeID {
					return nil, true
				}
			}
		}

	}
	return nil, false
}

func removElemSlice(elem string, list []string) []string {
	for index, val := range list {
		if elem == val {
			if len(list) > 1 {
				if index < len(list)-1 {
					list = append(list[0:index], list[index+1:]...)
				} else {
					list = list[0:index]
				}
			} else {
				list = nil
			}
		}
	}
	return list
}

func main() {
	/*
		list := []string{"kind-worker", "kind-worker2", "kind-control-plane"}
		list = removElemSlice("kind-control-plane", list)
		fmt.Printf("%+v\n", list)
	*/
	/*
		state, err := returnConfigMapKey("/home/user/.kube/config", "state2", "kured-brain", "kube-system")
		if err != nil {
			log.Printf("Error: %v\n", err)
		}
		fmt.Printf("%+v\n", state)
		state.Nodes = []string{"test-masters-0", "test-workers-0", "test-workers-1"}
		strState := PackConfigMapVals(state)
		fmt.Printf("STRSTATE: %+v\n", strState)
		strState = stripBracketsFromString(strState)
		fmt.Printf("STRSTATEPROCESSED: %+v\n", strState)
		err = setConfigMapKey("/home/user/.kube/config", "kured-brain", "kube-system", "state2", strState)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		err, _ = rebootRequired("node01", "/home/user/.kube/config", "kured-brain", "kube-system", "state2")
		if err != nil {
			fmt.Printf("%v\n", err)
		}
	*/

	config, err := clientcmd.BuildConfigFromFlags("", "/home/user/.kube/config")
	if err != nil {
		return
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	node, err := client.CoreV1().Nodes().Get(context.TODO(), "kind-worker", metav1.GetOptions{})
	fmt.Printf("%+v\n", node)
	for _, val := range node.Status.Conditions {
		if val.Type == "Ready" && val.Status == "True" && node.Spec.Unschedulable == false {
			fmt.Printf("READY WAY!!!")
		}
	}

}

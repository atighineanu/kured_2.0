package kuredbrain

import (
	"bytes"
	"context"
	"strings"
	"text/template"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
func ParseConfigMapVals(mapData string) State {
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

func StripBracketsFromString(state string) (processedState string) {
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

func SetConfigMapKey(client *kubernetes.Clientset, configMapName, namespace, stateKey, stringState string) (err error) {
	ctx := context.Background()
	configMap, err := client.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return
	}

	configMap.Data[stateKey] = stringState
	_, err = client.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
	if err != nil {
		return
	}
	return
}

func ReturnStringConfigMapKey(client *kubernetes.Clientset, key, configMapName, namespace string) (state string, err error) {
	ctx := context.Background()
	configMap, err := client.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return
	}
	state = configMap.Data[key]
	return
}

func ReturnConfigMapKey(client *kubernetes.Clientset, key, configMapName, namespace string) (state State, err error) {
	ctx := context.Background()
	configMap, err := client.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return
	}
	state = ParseConfigMapVals(configMap.Data[key])
	return
}

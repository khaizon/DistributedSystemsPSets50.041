package lib

import (
	"encoding/json"
	"fmt"
)

func (cm *CentralManager) log(args ...interface{}) {
	debug := args[0].(bool)
	mainString := args[1].(string)
	if (cm.Debug && debug) || !debug {
		fmt.Printf("CM %v: %v\n", cm.Id, fmt.Sprintf(mainString, args[2:]...))
	}
}

func (p *Processor) log(args ...interface{}) {
	debug := args[0].(bool)
	mainString := args[1].(string)
	if (p.Debug && debug) || !debug {
		fmt.Printf("%v: %v\n", p.Id, fmt.Sprintf(mainString, args[2:]...))
	}
}

func MessageArrayRemove(arr []Message, c int) []Message {
	for i := 0; i < len(arr); i++ {
		if arr[i].Sender == c {
			return append(arr[:i], arr[i+1:]...)
		}
	}
	return arr
}

func (s *State) Clone() (State, error) {
	currentStateJson, err := json.Marshal(s)
	if err != nil {
		return State{}, err
	}

	stateCopy := State{}
	if err = json.Unmarshal(currentStateJson, &stateCopy); err != nil {
		return State{}, err
	}

	return stateCopy, nil
}

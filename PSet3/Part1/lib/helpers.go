package lib

import "fmt"

func (c *CentralManager) log(args ...interface{}) {
	debug := args[0].(bool)
	mainString := args[1].(string)
	if (c.Debug && debug) || !debug {
		fmt.Printf("CM: %v\n", fmt.Sprintf(mainString, args[2:]...))
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
		if i == c {
			return append(arr[:i], arr[i+1:]...)
		}
	}
	return arr
}

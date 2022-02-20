package main

import "fmt"

type Person struct {
	name    string
	parents []Person
}

func (p Person) greet() {
	fmt.Println(p.name)
}

func (p *Person) rename(newName string) {
	p.name = newName
}

func main() {
	p := Person{name: "Juan"}
	p.greet()

	p.rename("Carlos")
	p.greet()
}

func rename(p Person, newName string) {
	p.name = newName
}

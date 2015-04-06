package main

import (
	"fmt"
)

type User struct {
	Name     string
	Password string
}

func main() {
	myVar := User{"Foo", "Bar"}
	fmt.Printf("myVar: %v", myVar)
}

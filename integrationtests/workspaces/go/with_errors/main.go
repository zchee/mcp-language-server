package main

import "fmt"

// FooBar is a simple function for testing
func FooBar() string {
	return "Hello, World!"
	fmt.Println("Unreachable code") // This is unreachable code
}

func main() {
	fmt.Println(FooBar())
}
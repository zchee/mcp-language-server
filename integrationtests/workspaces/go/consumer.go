package main

import "fmt"

// ConsumerFunction uses the helper function
func ConsumerFunction() {
	message := HelperFunction()
	fmt.Println(message)
}
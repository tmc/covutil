package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) > 1 {
		processArgs()
	} else {
		defaultBehavior()
	}
}

func processArgs() {
	for i, arg := range os.Args[1:] {
		if num, err := strconv.Atoi(arg); err == nil {
			processNumber(num)
		} else {
			processString(arg, i)
		}
	}
}

func processNumber(num int) {
	if num > 0 {
		fmt.Printf("Positive number: %d\n", num)
	} else if num < 0 {
		fmt.Printf("Negative number: %d\n", num)
	} else {
		fmt.Printf("Zero!\n")
	}
}

func processString(s string, index int) {
	if len(s) > 5 {
		fmt.Printf("Long string at %d: %s\n", index, s)
	} else {
		fmt.Printf("Short string at %d: %s\n", index, s)
	}
}

func defaultBehavior() {
	fmt.Println("Hello from simple program!")
	demonstrateFeatures()
}

func demonstrateFeatures() {
	fmt.Println("Features:")
	fmt.Println("- Number processing")
	fmt.Println("- String handling")
}
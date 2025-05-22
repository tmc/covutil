package main

import (
	"fmt"
	"runtime/coverage"
)

func main() {
	fmt.Println("SIMPLE TEST: main function called")
	fmt.Printf("SIMPLE TEST: Custom overlay active = %v\n", coverage.CUSTOM_OVERLAY_ACTIVE)
}

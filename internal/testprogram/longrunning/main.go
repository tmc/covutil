package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("LONGRUNNING: main function called")

	fmt.Println("LONGRUNNING: Sleeping for 10 seconds... (send SIGINT/Ctrl+C to test exit hook)")
	time.Sleep(10 * time.Second)
	fmt.Println("LONGRUNNING: Finished sleeping")
}

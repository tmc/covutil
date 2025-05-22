package pkga

import (
	"fmt"
	"os"
)

func Yo() {
	fmt.Fprintln(os.Stderr, "yo")
}

//go:build ignore

// Not part of the module. It is a fixture that `go run` compiles on its own,
// and without this it would be built by ./... as a second main package, and
// compiled concurrently with the test that runs it.
//
// A port in miniature: it asks something, then prints numbers it drew at
// random. Without a seed those numbers differ on every run, which is what makes
// a program like this impossible to script.
package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

func main() {
	fmt.Print("HOW MANY? ")

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		os.Exit(1)
	}

	count, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		os.Exit(1)
	}

	drawn := make([]string, count)
	for i := range drawn {
		drawn[i] = strconv.Itoa(rand.Intn(1000))
	}
	fmt.Println(strings.Join(drawn, " "))
}

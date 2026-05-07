package main

import (
	"encoding/json"
	"fmt"
	"os"

	"redshell/internal/marketplace"
)

func main() {
	svc := marketplace.NewService()
	list, err := svc.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(list)
}

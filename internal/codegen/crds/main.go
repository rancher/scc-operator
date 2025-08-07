package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	// Define controller-gen command and arguments
	cmdArgs := []string{
		"tool",
		"-modfile",
		"gotools/controller-gen/go.mod",
		"controller-gen",
		"crd:generateEmbeddedObjectMeta=true,allowDangerousTypes=false",
		"paths=./pkg/apis/...",
		"output:crd:dir=./pkg/crds/yaml/generated",
	}

	fmt.Printf("Executing command: go %s\n", strings.Join(cmdArgs, " "))
	runControllerGen(cmdArgs)

	fmt.Println("controller-gen command executed successfully.")

	// Remove empty CRD
	emptyCRDPath := "./pkg/crds/yaml/generated/_.yaml"
	if _, err := os.Stat(emptyCRDPath); err == nil {
		fmt.Printf("Removing empty CRD: %s\n", emptyCRDPath)
		if err := os.Remove(emptyCRDPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error removing empty CRD %s: %v\n", emptyCRDPath, err)
			os.Exit(1)
		}
		fmt.Println("Empty CRD removed successfully.")
	} else if !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error checking for empty CRD %s: %v\n", emptyCRDPath, err)
		os.Exit(1)
	} else {
		fmt.Println("No empty CRD found to remove.")
	}
}

func runControllerGen(cmdArgs []string) {
	cmd := exec.Command("go", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "controller-gen command failed: %v\n", err)
		os.Exit(1)
	}
}

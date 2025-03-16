package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/anhbkpro/jwt-blacklist-go/internal/utils"
)

func main() {
	// Define flags
	var passwordsFlag string
	flag.StringVar(&passwordsFlag, "p", "", "Comma-separated list of passwords to hash")
	flag.StringVar(&passwordsFlag, "passwords", "", "Comma-separated list of passwords to hash")

	var interactive bool
	flag.BoolVar(&interactive, "i", false, "Interactive mode to enter passwords")
	flag.BoolVar(&interactive, "interactive", false, "Interactive mode to enter passwords")

	// Parse flags
	flag.Parse()

	if interactive {
		runInteractiveMode()
		return
	}

	// Get passwords from flag
	if passwordsFlag == "" {
		fmt.Println("Error: No passwords provided. Use -p flag or -i for interactive mode.")
		fmt.Println("Example: ./hash -p \"password1,password2\"")
		flag.PrintDefaults()
		os.Exit(1)
	}

	passwords := strings.Split(passwordsFlag, ",")
	utils.PrintMultiplePasswordHashes(passwords...)
}

func runInteractiveMode() {
	fmt.Println("Password Hash Generator - Interactive Mode")
	fmt.Println("Enter passwords one per line. Press Ctrl+D (Unix) or Ctrl+Z (Windows) when done.")

	var passwords []string
	var password string

	for {
		fmt.Print("Password: ")
		_, err := fmt.Scanln(&password)
		if err != nil {
			break
		}

		if password != "" {
			passwords = append(passwords, password)
		}
	}

	if len(passwords) == 0 {
		fmt.Println("No passwords entered.")
		return
	}

	utils.PrintMultiplePasswordHashes(passwords...)
}

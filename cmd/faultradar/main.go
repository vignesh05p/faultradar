package main

import (
	"fmt"
	"os"

	"faultradar/internal/app"
	"faultradar/internal/report"
	"faultradar/internal/system"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]
	switch subcommand {
	case "doctor":
		if len(os.Args) == 3 {
			switch os.Args[2] {
			case "--json":
				runDoctor(true)
			case "--help":
				printHelp()
				os.Exit(0)
			default:
				fmt.Printf("Unknown flag: %s\n\n", os.Args[2])
				printUsage()
				os.Exit(1)
			}
		} else if len(os.Args) == 2 {
			runDoctor(false)
		} else {
			fmt.Println("Too many arguments for doctor command.")
			printUsage()
			os.Exit(1)
		}

	case "version":
		if len(os.Args) != 2 {
			printUsage()
			os.Exit(1)
		}
		fmt.Printf("faultradar %s\n", version)
		os.Exit(0)

	case "help", "--help", "-h":
		printHelp()
		os.Exit(0)

	default:
		fmt.Printf("Unknown command: %s\n\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func runDoctor(jsonMode bool) {
	fs := system.RealFileSystem{}
	runner := system.RealCommandRunner{}

	config, configFindings := app.LoadConfig(fs)

	doc := app.Doctor{
		Config: config,
		Runner: runner,
		FS:     fs,
	}

	findings := doc.Run()

	if len(configFindings) > 0 {
		findings = append(configFindings, findings...)
	}

	if jsonMode {
		err := report.PrintJSON(os.Stdout, findings)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
			os.Exit(3)
		}
	} else {
		report.PrintHuman(os.Stdout, version, findings)
	}

	os.Exit(app.ExitCode(findings))
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Unknown command or invalid usage.\n\n")
	printHelp()
}

func printHelp() {
	fmt.Printf("FaultRadar v%s\n\n", version)
	fmt.Println("Usage:")
	fmt.Println("  faultradar doctor [--json]")
	fmt.Println("  faultradar version")
	fmt.Println("  faultradar help")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  doctor    Run read-only Linux health checks")
	fmt.Println("  version   Print version")
	fmt.Println("  help      Show help")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --json    Print machine-readable JSON for doctor output")
}

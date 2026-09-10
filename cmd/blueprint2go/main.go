// blueprint2go generates Go types from Aiken's plutus.json (CIP-0057 Plutus Blueprint).
//
// Usage:
//
//	blueprint2go plutus.json -o types.go
//	blueprint2go plutus.json -o types.go -p mypackage
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/SundaeSwap-finance/blueprint2go/pkg/blueprint"
)

func main() {
	var (
		outfile     string
		packageName string
	)

	flag.StringVar(&outfile, "o", "", "Output file path (required)")
	flag.StringVar(&outfile, "outfile", "", "Output file path (required)")
	flag.StringVar(&packageName, "p", "contracts", "Go package name")
	flag.StringVar(&packageName, "package", "contracts", "Go package name")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <plutus.json>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generate Go types from Aiken's CIP-0057 Plutus Blueprint.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -o types.go plutus.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -o types.go -p mypackage plutus.json\n", os.Args[0])
	}

	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: plutus.json file is required")
		flag.Usage()
		os.Exit(1)
	}

	if outfile == "" {
		fmt.Fprintln(os.Stderr, "Error: output file (-o) is required")
		flag.Usage()
		os.Exit(1)
	}

	infile := flag.Arg(0)

	// Load blueprint
	bp, err := blueprint.LoadBlueprint(infile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading blueprint: %v\n", err)
		os.Exit(1)
	}

	// Generate code
	gen := blueprint.NewGenerator(bp, blueprint.GeneratorOptions{
		PackageName: packageName,
	})

	code, err := gen.Generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating code: %v\n", err)
		os.Exit(1)
	}

	// Write output
	if err := os.WriteFile(outfile, []byte(code), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s from %s\n", outfile, infile)
}

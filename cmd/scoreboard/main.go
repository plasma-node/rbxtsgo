// Command scoreboard prints how close rbxtsgo is to JS roblox-ts@3.0.0, using the goldens in testdata/differential.
// Run it through scripts/scoreboard.sh, which also does the steps that need the binary. See
// testdata/differential/README.md.
//
//	scoreboard [-checkers n] [-v] [-only corpus,corpus] [-report file]
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/microsoft/TypeScript/tsc/rbxtsgo/internal/difftest"
)

func main() {
	checkers := flag.Int("checkers", 0, "number of checkers and workers")
	verbose := flag.Bool("v", false, "print the first difference of every failure")
	only := flag.String("only", "", "comma-separated corpus names")
	reportPath := flag.String("report", "", "also write every failure to this file")
	flag.Parse()

	cwd, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	root, err := difftest.RepoRoot(cwd)
	if err != nil {
		fatal(err)
	}
	options := difftest.Options{Checkers: *checkers}
	if *only != "" {
		options.Only = strings.Split(*only, ",")
	}
	started := time.Now()
	report, err := difftest.Run(root, options)
	if err != nil {
		fatal(err)
	}
	elapsed := time.Since(started)

	var detail strings.Builder
	corpora := map[string]bool{}
	var order []string
	for _, result := range report.Results {
		if !corpora[result.Corpus] {
			corpora[result.Corpus] = true
			order = append(order, result.Corpus)
		}
		if !result.Pass {
			fmt.Fprintf(&detail, "FAIL %s\n%s\n\n", result.Key(), indent(result.Detail))
		}
	}
	if *verbose {
		fmt.Print(detail.String())
	}
	if *reportPath != "" {
		if err := os.WriteFile(*reportPath, []byte(detail.String()), 0o666); err != nil {
			fatal(err)
		}
	}

	fmt.Printf("%-18s %14s %14s %14s\n", "corpus", "files", "diagnostics", "reprints")
	for _, name := range order {
		var outputPass, outputTotal, diagnosticPass, diagnosticTotal, reprintPass, reprintTotal int
		for _, result := range report.Results {
			if result.Corpus != name {
				continue
			}
			if result.Kind == "output" {
				outputTotal++
				if result.Pass {
					outputPass++
				}
			} else if result.Kind == "reprint" {
				reprintTotal++
				if result.Pass {
					reprintPass++
				}
			} else {
				diagnosticTotal++
				if result.Pass {
					diagnosticPass++
				}
			}
		}
		fmt.Printf("%-18s %14s %14s %14s\n", name, fmt.Sprintf("%d/%d", outputPass, outputTotal), fmt.Sprintf("%d/%d", diagnosticPass, diagnosticTotal), fmt.Sprintf("%d/%d", reprintPass, reprintTotal))
	}
	outputPass, outputTotal := report.Counts("output")
	diagnosticPass, diagnosticTotal := report.Counts("diagnostics")
	reprintPass, reprintTotal := report.Counts("reprint")
	fmt.Printf("%-18s %14s %14s %14s\n", "total", fmt.Sprintf("%d/%d", outputPass, outputTotal), fmt.Sprintf("%d/%d", diagnosticPass, diagnosticTotal), fmt.Sprintf("%d/%d", reprintPass, reprintTotal))
	fmt.Printf("in-process compile and compare: %.1fs\n", elapsed.Seconds())

	if summary := report.PanicSummary(); len(summary) > 0 {
		fmt.Println("\ninternal errors by count (the work list):")
		for _, line := range summary {
			fmt.Println(line)
		}
	}
}

func indent(text string) string {
	return "    " + strings.ReplaceAll(text, "\n", "\n    ")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

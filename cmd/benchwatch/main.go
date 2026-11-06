// Command benchwatch measures save-to-output latency of watch mode. scripts/bench-watch.sh is the way to run it.
//
// It starts the watcher, waits for the first build, then edits one file N times. Each edit adds a statement, so the
// output has to change, and the clock runs from just before the write until the output file holds the new bytes. The
// original file is put back at the end, whatever happens, and the watcher is killed and waited for.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/tsbridge"
)

var timingField = regexp.MustCompile(`(debounce|program|copy|prepare|plugin|check|lower|write|declarations) ([0-9.]+)ms`)

func main() { os.Exit(run()) }

func run() int {
	binary := flag.String("bin", "", "path to rbxtsgo")
	projectPath := flag.String("project", ".", "project folder or tsconfig")
	file := flag.String("file", "", "source file to edit")
	count := flag.Int("n", 20, "number of edits")
	waitFile := flag.String("wait", "", "output file to wait on, default: the output of -file")
	template := flag.String("append", `print("__rbxtsgo_bench_%d");`, "text each edit adds, with %d for the edit number. The output has to change")
	settle := flag.Duration("settle", 300*time.Millisecond, "pause after each rebuild")
	checkers := flag.Int("checkers", 0, "pool size to pass on")
	flag.Parse()
	if *binary == "" || *file == "" {
		fmt.Fprintln(os.Stderr, "-bin and -file are required")
		return 2
	}
	sourceFile, err := filepath.Abs(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	original, err := os.ReadFile(sourceFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	restore := sync.OnceFunc(func() {
		if err := os.WriteFile(sourceFile, original, 0o666); err != nil {
			fmt.Fprintln(os.Stderr, "COULD NOT RESTORE", sourceFile, err)
		}
	})
	defer restore()
	interrupt, stopInterrupt := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stopInterrupt()
	go func() {
		<-interrupt.Done()
		restore()
	}()

	output := *waitFile
	if output == "" {
		output, err = outputPath(*projectPath, sourceFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "could not work out the output path. Pass -wait:", err)
			return 1
		}
	}

	args := []string{"-w", "--verbose", "-p", *projectPath}
	if *checkers > 0 {
		args = append(args, "--checkers", strconv.Itoa(*checkers))
	}
	cmd := exec.Command(*binary, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cmd.Stderr = os.Stderr
	started := time.Now()
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	stopped := sync.OnceFunc(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	defer stopped()

	// Every cycle prints one "Found" line and then one timing line. Only the first "Found" line gets used, and nothing
	// may block the reader, or the watcher blocks on its stdout.
	timing := make(chan string, 64)
	found := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1<<20), 1<<26)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "Watching for file changes.") {
				select {
				case found <- line:
				default:
				}
			}
			if strings.HasPrefix(line, "[timing]") {
				timing <- line
			}
		}
		close(timing)
	}()
	waitTiming := func(limit time.Duration) (string, bool) {
		select {
		case line, ok := <-timing:
			return line, ok
		case <-time.After(limit):
			return "", false
		case <-interrupt.Done():
			return "", false
		}
	}

	initial, ok := waitTiming(5 * time.Minute)
	if !ok {
		fmt.Fprintln(os.Stderr, "the watcher never finished its first build")
		return 1
	}
	fmt.Printf("first build: %d ms, %s\n", time.Since(started).Milliseconds(), stripColor(<-found))
	fmt.Println(initial)
	var firstMemory, lastMemory, peakMemory uint64
	if current, peak, ok := memoryOf(cmd.Process.Pid); ok {
		firstMemory, lastMemory, peakMemory = current, current, peak
	}

	latencies := make([]time.Duration, 0, *count)
	phases := map[string][]float64{}
	var last string
	for i := range *count {
		time.Sleep(*settle)
		// Reports from a save that landed in two cycles.
		for drained := false; !drained; {
			select {
			case <-timing:
			default:
				drained = true
			}
		}
		before, _ := os.ReadFile(output)
		text := string(original) + "\n" + fmt.Sprintf(*template, i) + "\n"
		begin := time.Now()
		if err := os.WriteFile(sourceFile, []byte(text), 0o666); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		deadline := begin.Add(2 * time.Minute)
		for {
			now, err := os.ReadFile(output)
			if err == nil && string(now) != string(before) {
				latencies = append(latencies, time.Since(begin))
				break
			}
			if time.Now().After(deadline) || interrupt.Err() != nil {
				fmt.Fprintf(os.Stderr, "edit %d: %s did not change. Does the edit compile, and does it change the output?\n", i, output)
				return 1
			}
			time.Sleep(time.Millisecond)
		}
		line, ok := waitTiming(2 * time.Minute)
		if !ok {
			fmt.Fprintln(os.Stderr, "the watcher stopped reporting")
			return 1
		}
		last = line
		if current, peak, ok := memoryOf(cmd.Process.Pid); ok {
			lastMemory, peakMemory = current, peak
		}
		for _, match := range timingField.FindAllStringSubmatch(line, -1) {
			value, _ := strconv.ParseFloat(match[2], 64)
			phases[match[1]] = append(phases[match[1]], value)
		}
	}

	if peakMemory > 0 {
		fmt.Printf("memory: %d MB after the first build, %d MB after the last edit, peak %d MB\n", firstMemory>>20, lastMemory>>20, peakMemory>>20)
	}
	restore()
	stopped()

	sorted := slices.Clone(latencies)
	slices.Sort(sorted)
	fmt.Printf("edits: %d, save to output, uncontrolled: median %.1f ms, min %.1f ms, max %.1f ms\n", len(sorted), msOf(sorted[len(sorted)/2]), msOf(sorted[0]), msOf(sorted[len(sorted)-1]))
	fmt.Print("phase medians (ms):")
	for _, name := range []string{"debounce", "program", "copy", "prepare", "plugin", "check", "lower", "write", "declarations"} {
		if list := phases[name]; len(list) > 0 {
			slices.Sort(list)
			fmt.Printf(" %s %.1f", name, list[len(list)/2])
		}
	}
	fmt.Println()
	if last != "" {
		fmt.Println("last:", last)
	}
	return 0
}

func msOf(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

var colorCode = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripColor(text string) string { return colorCode.ReplaceAllString(text, "") }

func outputPath(projectPath, sourceFile string) (string, error) {
	tsconfig, err := filepath.Abs(projectPath)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(tsconfig); err == nil && info.IsDir() {
		tsconfig = filepath.Join(tsconfig, "tsconfig.json")
	}
	data, err := project.CreateProjectData(tsconfig, shared.DefaultProjectOptions())
	if err != nil {
		return "", err
	}
	parsed, err := project.GetParsedCommandLine(data)
	if err != nil {
		return "", err
	}
	p := tsbridge.NewProject(parsed, tsbridge.ProjectOptions{})
	translator := project.CreatePathTranslator(p, data)
	return translator.GetOutputPath(filepath.ToSlash(sourceFile)), nil
}

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const version = "0.4.2-preview.1"

type listFlag []string

func (f *listFlag) String() string { return strings.Join(*f, ",") }
func (f *listFlag) Set(v string) error {
	if strings.TrimSpace(v) == "" {
		return fmt.Errorf("empty argument")
	}
	*f = append(*f, v)
	return nil
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "BeforeUp: check a Docker Compose project against the local host before starting it.")
	fmt.Fprintln(w, "Usage: beforeup check [-f compose.yaml] [--env-file .env] [--profile NAME] [--format text|json] [--min-free-gib N] [--strict]")
	fmt.Fprintln(w, "       beforeup compare [-f compose.yaml] [--format text|json] [--strict] (compare against running services)")
	fmt.Fprintln(w, "       beforeup version")
	fmt.Fprintln(w, "No containers are created or started by BeforeUp.")
}
func main() { os.Exit(start(os.Args[1:], os.Stdout, os.Stderr)) }
func start(args []string, out, errw io.Writer) int {
	if len(args) == 0 {
		usage(errw)
		return 2
	}
	switch args[0] {
	case "version":
		fmt.Fprintln(out, version)
		return 0
	case "help", "-h", "--help":
		usage(out)
		return 0
	case "check", "compare":
	default:
		usage(errw)
		return 2
	}
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(errw)
	var files, envFiles, profiles listFlag
	var format string
	var minFree uint64
	var strict bool
	fs.Var(&files, "f", "Compose file (repeatable)")
	fs.Var(&files, "file", "Compose file (repeatable)")
	fs.Var(&envFiles, "env-file", "Environment file for Compose (repeatable)")
	fs.Var(&profiles, "profile", "Compose profile (repeatable)")
	fs.StringVar(&format, "format", "text", "Output: text or json")
	fs.Uint64Var(&minFree, "min-free-gib", 2, "Warn when Docker filesystem has less than N GiB free; 0 disables this check")
	fs.BoolVar(&strict, "strict", false, "Treat warnings as a nonzero exit status (for CI)")
	if e := fs.Parse(args[1:]); e != nil {
		return 2
	}
	if fs.NArg() > 0 || (format != "text" && format != "json") {
		fmt.Fprintln(errw, "Unknown argument or output format. Run 'beforeup help'.")
		return 2
	}
	report, scanErr := inspect(context.Background(), Options{Files: []string(files), EnvFiles: []string(envFiles), Profiles: []string(profiles), MinFreeGiB: minFree, Compare: args[0] == "compare"}, realHost())
	if format == "json" {
		b, _ := json.MarshalIndent(report, "", "  ")
		fmt.Fprintln(out, string(b))
	} else {
		if report.Project != "" {
			fmt.Fprintf(out, "BeforeUp / %s\n\n", report.Project)
		} else {
			fmt.Fprintln(out, "BeforeUp")
		}
		for _, f := range report.Findings {
			fmt.Fprintf(out, "%-7s %-15s %s\n", strings.ToUpper(f.Level), f.ID, f.Message)
			if f.Service != "" {
				fmt.Fprintf(out, "        Service: %s\n", f.Service)
			}
			if f.Advice != "" {
				fmt.Fprintf(out, "        Next: %s\n", f.Advice)
			}
		}
		fmt.Fprintf(out, "\n%d blocker(s), %d warning(s), %d passed\n", report.Summary.Blockers, report.Summary.Warnings, report.Summary.Passed)
	}
	return reportExitCode(report, scanErr, strict)
}
func reportExitCode(report Report, scanErr error, strict bool) int {
	if scanErr != nil {
		return 2
	}
	if report.Summary.Blockers > 0 || strict && report.Summary.Warnings > 0 {
		return 1
	}
	return 0
}

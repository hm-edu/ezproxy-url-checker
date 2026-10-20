package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const (
	urlsFile    = "urls.txt"
	hostsFile   = "hosts.txt"
	domainsFile = "domains.txt"
)

// listFiles are the files written by generate, in output order.
var listFiles = []string{urlsFile, hostsFile, domainsFile}

// directiveLists maps the (lower-cased) EZProxy directives that feed a list to
// that list. DJ/DomainJavascript is deliberately not included.
var directiveLists = map[string]string{
	"u":              urlsFile,
	"url":            urlsFile,
	"h":              hostsFile,
	"host":           hostsFile,
	"hj":             hostsFile,
	"hostjavascript": hostsFile,
	"d":              domainsFile,
	"domain":         domainsFile,
}

// hostPattern reduces a directive value to its host part: an optional scheme is
// dropped and so is everything from the first slash on, so
// "https://example.com/login?x=1" becomes "example.com".
var hostPattern = regexp.MustCompile(`(?i)^(?:https?://)?([^/]+)`)

// runGenerate implements the `generate` subcommand.
func runGenerate(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "config", "directory to write urls.txt, hosts.txt and domains.txt to")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: ezproxy-url-checker generate [-out DIR] <ezproxy-dir>")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Extracts the URL, Host and Domain directives from the EZProxy config.txt in")
		fmt.Fprintln(stderr, "<ezproxy-dir> (and the files it includes) into the lists served by this program.")
		fmt.Fprintln(stderr)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return errors.New("need to pass the directory containing the EZProxy config.txt")
	}

	return generate(fs.Arg(0), *out, stdout)
}

// generate reads the EZProxy config.txt in dir, follows its IncludeFile
// directives and writes the sorted, de-duplicated lists to out.
func generate(dir, out string, log io.Writer) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	g := &generator{
		dir:   dir,
		seen:  map[string]bool{},
		lists: map[string]map[string]struct{}{},
		log:   log,
	}
	for _, name := range listFiles {
		g.lists[name] = map[string]struct{}{}
	}
	if err := g.processFile("config.txt"); err != nil {
		return err
	}

	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	for _, name := range listFiles {
		if err := writeList(filepath.Join(out, name), g.lists[name]); err != nil {
			return err
		}
	}

	fmt.Fprintf(log, "Wrote %d urls, %d hosts and %d domains to %s\n",
		len(g.lists[urlsFile]), len(g.lists[hostsFile]), len(g.lists[domainsFile]), out)
	return nil
}

type generator struct {
	dir   string                         // EZProxy directory; IncludeFile paths are relative to it
	seen  map[string]bool                // files already processed, guards against include loops
	lists map[string]map[string]struct{} // list file name -> set of values
	log   io.Writer
}

// processFile collects the directives of one EZProxy config file and recurses
// into the files it includes.
func (g *generator) processFile(name string) error {
	path := name
	if !filepath.IsAbs(path) {
		path = filepath.Join(g.dir, name)
	}
	if g.seen[path] {
		return nil
	}
	g.seen[path] = true
	fmt.Fprintln(g.log, path)

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// Fields also swallows a trailing \r, so CRLF files work as well.
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}

		directive := strings.ToLower(fields[0])
		if directive == "includefile" {
			if err := g.processFile(fields[1]); err != nil {
				return err
			}
			continue
		}

		list, ok := directiveLists[directive]
		if !ok {
			continue
		}
		if value := directiveValue(fields[1:]); value != "" {
			g.lists[list][value] = struct{}{}
		}
	}
	return scanner.Err()
}

// directiveValue returns the host part of the first argument of a directive
// that is not a qualifier such as the "-Refresh" in "URL -Refresh https://...".
func directiveValue(args []string) string {
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if m := hostPattern.FindStringSubmatch(arg); m != nil {
			return m[1]
		}
		return arg
	}
	return ""
}

// writeList writes the values of set to path, one per line, sorted.
func writeList(path string, set map[string]struct{}) error {
	var b strings.Builder
	for _, v := range slices.Sorted(maps.Keys(set)) {
		b.WriteString(v)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

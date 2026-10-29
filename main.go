package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	var outputFile string
	var noGitignore bool

	flag.StringVar(&outputFile, "o", "output.txt", "Archivo de salida")
	flag.BoolVar(&noGitignore, "no-gitignore", false, "No leer .gitignore (el .spignore se sigue aplicando siempre)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Uso: %s [opciones] <path_carpeta>\n\nOpciones:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	rootPath := flag.Arg(0)
	info, err := os.Stat(rootPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s no es un directorio\n", rootPath)
		os.Exit(1)
	}

	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolviendo ruta raíz: %v\n", err)
		os.Exit(1)
	}

	// Cargar patrones de .spignore (siempre)
	var spignorePatterns []gitignorePattern
	spignorePath := filepath.Join(absRoot, ".spignore")
	if data, err := os.ReadFile(spignorePath); err == nil {
		spignorePatterns = parseGitignore(string(data))
		fmt.Fprintf(os.Stderr, "Cargados %d patrones de .spignore\n", len(spignorePatterns))
	}

	// Cargar patrones de .gitignore (salvo que se desactive)
	var gitignorePatterns []gitignorePattern
	if !noGitignore {
		gitignorePath := filepath.Join(absRoot, ".gitignore")
		if data, err := os.ReadFile(gitignorePath); err == nil {
			gitignorePatterns = parseGitignore(string(data))
			fmt.Fprintf(os.Stderr, "Cargados %d patrones de .gitignore\n", len(gitignorePatterns))
		}
	}

	matchesPatterns := func(patterns []gitignorePattern, rel string, isDir bool) bool {
		matched := false
		for _, p := range patterns {
			if p.match(rel, isDir) {
				matched = !p.negate
			}
		}
		return matched
	}

	isExcluded := func(rel string, isDir bool) bool {
		if matchesPatterns(spignorePatterns, rel, isDir) {
			return true
		}
		if matchesPatterns(gitignorePatterns, rel, isDir) {
			return true
		}
		return false
	}

	// Recolectar archivos
	var files []string
	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		rel, relErr := filepath.Rel(absRoot, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}

		// Exclusión por .spignore o .gitignore
		if isExcluded(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		files = append(files, path)
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error recorriendo directorio: %v\n", err)
		os.Exit(1)
	}

	sort.Strings(files)

	out, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creando archivo de salida: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	separator := strings.Repeat("-", 80)
	first := true
	written := 0

	for _, file := range files {
		relPath, err := filepath.Rel(absRoot, file)
		if err != nil {
			continue
		}
		relPath = filepath.ToSlash(relPath)

		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Advertencia: no se pudo leer %s: %v\n", file, err)
			continue
		}

		if isBinary(content) {
			continue
		}

		if !first {
			fmt.Fprintln(out, separator)
		}
		first = false

		fmt.Fprintln(out, relPath)
		out.Write(content)
		if len(content) > 0 && content[len(content)-1] != '\n' {
			fmt.Fprintln(out)
		}
		written++
	}

	fmt.Printf("Generado: %s (%d archivos)\n", outputFile, written)
}

// ---- gitignore-style patterns ----

type gitignorePattern struct {
	original string
	negate   bool
	anchored bool
	dirOnly  bool
	hasSlash bool
	segments []string
}

func parseGitignore(content string) []gitignorePattern {
	var patterns []gitignorePattern
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimRight(line, "\r")
		line = strings.TrimRight(line, " ")

		if line == "" {
			continue
		}

		p := gitignorePattern{original: line}

		if strings.HasPrefix(line, "!") {
			p.negate = true
			line = line[1:]
		}

		line = strings.TrimPrefix(line, "\\")

		if strings.HasSuffix(line, "/") {
			p.dirOnly = true
			line = strings.TrimSuffix(line, "/")
		}

		if strings.HasPrefix(line, "/") {
			p.anchored = true
			line = strings.TrimPrefix(line, "/")
		}

		if line == "" {
			continue
		}

		p.hasSlash = strings.Contains(line, "/")
		p.segments = strings.Split(line, "/")
		patterns = append(patterns, p)
	}

	return patterns
}

func (p gitignorePattern) match(rel string, isDir bool) bool {
	if p.dirOnly && !isDir {
		return false
	}

	parts := strings.Split(rel, "/")

	if !p.hasSlash && !p.anchored {
		if matchSegment(p.segments[0], parts[len(parts)-1]) {
			return true
		}
		for _, seg := range parts {
			if matchSegment(p.segments[0], seg) {
				return true
			}
		}
		return false
	}

	return matchSegmentsFromRoot(p.segments, parts)
}

func matchSegment(pattern, s string) bool {
	if !strings.ContainsAny(pattern, "*?[") {
		return pattern == s
	}
	ok, err := filepath.Match(pattern, s)
	if err != nil {
		return false
	}
	return ok
}

func matchSegmentsFromRoot(pattern, parts []string) bool {
	var match func(pi, si int) bool
	match = func(pi, si int) bool {
		if pi == len(pattern) {
			return true
		}
		if si == len(parts) {
			if pattern[pi] == "**" {
				return match(pi+1, si)
			}
			return false
		}

		if pattern[pi] == "**" {
			if match(pi+1, si) {
				return true
			}
			return match(pi, si+1)
		}

		if matchSegment(pattern[pi], parts[si]) {
			return match(pi+1, si+1)
		}
		return false
	}

	return match(0, 0)
}

func isBinary(data []byte) bool {
	limit := len(data)
	if limit > 8192 {
		limit = 8192
	}
	for i := 0; i < limit; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LessCompiler handles LESS to CSS compilation using lessc (Node.js)
type LessCompiler struct {
	magentoRoot string
	verbose     bool
	lesscPath   string
}

// NewLessCompiler creates a new LESS compiler instance
func NewLessCompiler(magentoRoot string, verbose bool) (*LessCompiler, error) {
	// Find lessc in PATH
	lesscPath, err := exec.LookPath("lessc")
	if err != nil {
		return nil, fmt.Errorf("lessc not found in PATH. Install with: npm install -g less")
	}

	return &LessCompiler{
		magentoRoot: magentoRoot,
		verbose:     verbose,
		lesscPath:   lesscPath,
	}, nil
}

// CompileEmailCSS compiles the email LESS files to CSS for a given theme/locale/area
func (lc *LessCompiler) CompileEmailCSS(stagingDir, destDir string) error {
	// Email LESS files to compile
	emailFiles := []string{
		"email.less",
		"email-inline.less",
		"email-fonts.less",
	}

	for _, lessFileName := range emailFiles {
		sourcePath := filepath.Join(stagingDir, "css", lessFileName)

		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			if lc.verbose {
				fmt.Printf("    ⊘ %s not found\n", lessFileName)
			}
			continue
		}

		// Output CSS file path
		cssFileName := strings.TrimSuffix(lessFileName, ".less") + ".css"
		cssPath := filepath.Join(destDir, "css", cssFileName)

		// Ensure css directory exists
		os.MkdirAll(filepath.Join(destDir, "css"), 0755)

		// Compile LESS to CSS using lessc
		if err := lc.compileLessFile(sourcePath, cssPath, stagingDir); err != nil {
			if lc.verbose {
				fmt.Printf("    ✗ Failed to compile %s: %v\n", lessFileName, err)
			}
			continue
		}

		if lc.verbose {
			fmt.Printf("    ✓ Compiled %s → css/%s\n", lessFileName, cssFileName)
		}
	}

	return nil
}

// compileLessFile compiles a single LESS file to CSS using lessc
func (lc *LessCompiler) compileLessFile(sourcePath, destPath, stagingDir string) error {
	// Build include paths for @import resolution
	includePaths := []string{
		stagingDir,
		filepath.Join(stagingDir, "css"),
		filepath.Join(stagingDir, "css", "source"),
		filepath.Join(stagingDir, "css", "source", "lib"),
	}

	// Build lessc command with compression to match Magento's minified output
	args := []string{
		sourcePath,
		destPath,
		"--include-path=" + strings.Join(includePaths, ":"),
		"--compress",
	}

	cmd := exec.Command(lc.lesscPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("lessc failed: %v\nOutput: %s", err, string(output))
	}

	// Verify output file was created and has content
	info, err := os.Stat(destPath)
	if err != nil {
		return fmt.Errorf("output file not created: %w", err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("output file is empty")
	}

	return nil
}

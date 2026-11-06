package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// gstyle-allow(side_effect_density): Explicit subprocess/file boundary. Only a completed rstyle run becomes a viewer snapshot.
func captureRstyle(root, executable string) (rstyleStore, error) {
	temp, err := os.MkdirTemp("", "gstyle-rstyle-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(temp)
	reportPath := filepath.Join(temp, "report.json")
	runPath := filepath.Join(temp, "run")
	if err := runRstyle(root, executable, runPath, reportPath); err != nil {
		return nil, err
	}
	return loadRstyle(runPath, reportPath)
}

// gstyle-allow(side_effect_density): Executes the requested rstyle backend. Exit 1 is a completed analysis with findings; all other failures remain errors.
func runRstyle(root, executable, runPath, reportPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	output, err := os.Create(reportPath)
	if err != nil {
		return err
	}
	defer output.Close()
	command := exec.CommandContext(ctx, executable, "--cpg-compression", "none", "go", "--source", root, "--out", runPath, "--format", "json", "--filter", "check,patterns,flow")
	var stderr bytes.Buffer
	command.Stdout, command.Stderr = output, &stderr
	err = command.Run()
	if completedRstyle(err) {
		return nil
	}
	return fmt.Errorf("rstyle backend failed: %w: %s (install current rstyle and rstyle-go; use -rstyle for an explicit runner)", err, stderr.String())
}
func completedRstyle(err error) bool {
	if err == nil {
		return true
	}
	var exit *exec.ExitError
	return errors.As(err, &exit) && exit.ExitCode() == 1
}

// gstyle-allow(side_effect_density): Reads bounded files from the private completed run; rstyle validates all semantic facts before publishing it.
func loadRstyle(runPath, reportPath string) (rstyleStore, error) {
	var report rstyleReport
	if err := readRstyleReport(reportPath, &report); err != nil {
		return nil, err
	}

	seals, err := rstyleManifest(runPath)
	if err != nil {
		return nil, err
	}
	store := rstyleStore{}
	for _, seal := range seals {
		if err := loadRstyleUnit(store, runPath, seal, report); err != nil {
			return nil, err
		}
	}
	return store, nil
}

type rstyleSeal struct {
	File            string `json:"file"`
	Encoding        string `json:"encoding"`
	Bytes           int    `json:"bytes"`
	Checksum        string `json:"checksum"`
	DecodedBytes    int    `json:"decoded_bytes"`
	DecodedChecksum string `json:"decoded_checksum"`
}

// gstyle-allow(side_effect_density): Reads a single sealed JSON unit, rejecting paths, corruption and unsupported encodings.
func loadRstyleUnit(store rstyleStore, runPath string, seal rstyleSeal, report rstyleReport) error {
	if filepath.Base(seal.File) != seal.File || seal.Encoding != "json" {
		return fmt.Errorf("rstyle capture requires single-name plain JSON documents")
	}
	data, err := readSource(filepath.Join(runPath, "documents"), seal.File)
	if err != nil {
		return err
	}
	if err := seal.verify(data); err != nil {
		return err
	}
	var doc cpgDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}
	return store.add(doc, report)
}

// gstyle-allow(side_effect_density): Decodes one bounded JSON file; trailing values are rejected.
func decodeRstyle(path string, value any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 512<<20))
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("decode rstyle %s: %w", path, err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("rstyle %s has trailing data", path)
	}
	return nil
}

func (seal rstyleSeal) verify(data []byte) error {
	sum := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
	if seal.Bytes != len(data) || seal.DecodedBytes != len(data) || seal.Checksum != sum || seal.DecodedChecksum != sum {
		return fmt.Errorf("rstyle document seal mismatch: %s", seal.File)
	}
	return nil
}

// gstyle-allow(side_effect_density): Reads and validates the complete analysis envelope.
func readRstyleReport(path string, report *rstyleReport) error {
	if err := decodeRstyle(path, report); err != nil {
		return err
	}
	return report.validate()
}

// gstyle-allow(side_effect_density): Reads the run's completion manifest before any graph is exposed.
func rstyleManifest(runPath string) ([]rstyleSeal, error) {
	var manifest struct {
		Schema    string       `json:"schema"`
		Documents []rstyleSeal `json:"documents"`
	}
	if err := decodeRstyle(filepath.Join(runPath, "documents", "documents.manifest.json"), &manifest); err != nil {
		return nil, err
	}
	if manifest.Schema != "rstyle.cpg-documents.manifest/2" || len(manifest.Documents) == 0 {
		return nil, fmt.Errorf("rstyle capture has no supported completed manifest")
	}
	return manifest.Documents, nil
}

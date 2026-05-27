package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/konveyor/ai-rule-gen/internal/groundtruth"
	"gopkg.in/yaml.v3"
)

func main() {
	oldArtifact := flag.String("old-artifact", "", "Old artifact coordinate (groupId:artifactId:version)")
	newArtifact := flag.String("new-artifact", "", "New artifact coordinate (groupId:artifactId:version)")
	output := flag.String("output", "", "Output path for ground_truth.yaml (default: stdout)")
	japicmpJar := flag.String("japicmp-jar", "", "Path to japicmp standalone JAR (auto-downloaded if not set)")
	mergePath := flag.String("merge", "", "Existing ground_truth.yaml to merge with")
	guideURL := flag.String("guide-url", "", "Migration guide URL for the output file")
	flag.Parse()

	if *oldArtifact == "" || *newArtifact == "" {
		fmt.Fprintln(os.Stderr, "Usage: ground-truth --old-artifact G:A:V --new-artifact G:A:V [--output path] [--merge path]")
		os.Exit(1)
	}

	oldCoord, err := groundtruth.ParseCoord(*oldArtifact)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	newCoord, err := groundtruth.ParseCoord(*newArtifact)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	workDir, err := os.MkdirTemp("", "ground-truth-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(workDir)

	fmt.Fprintf(os.Stderr, "Downloading old artifact: %s\n", *oldArtifact)
	oldJar, err := groundtruth.DownloadJAR(oldCoord, workDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Downloading new artifact: %s\n", *newArtifact)
	newJar, err := groundtruth.DownloadJAR(newCoord, workDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Ensuring japicmp is available...\n")
	japicmp, err := groundtruth.EnsureJapicmp(*japicmpJar)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	diffXML := filepath.Join(workDir, "diff.xml")
	fmt.Fprintf(os.Stderr, "Running japicmp...\n")
	if err := groundtruth.RunJapicmp(japicmp, oldJar, newJar, diffXML); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	changes, err := groundtruth.ParseJapicmpXML(diffXML)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Found %d breaking API changes\n", len(changes))

	entries := groundtruth.ConvertChanges(changes)

	gt := &groundtruth.GroundTruth{
		SchemaVersion: 1,
		GuideURL:      *guideURL,
		Entries:       entries,
	}

	if *mergePath != "" {
		existing, err := groundtruth.ReadGroundTruth(*mergePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not read merge file: %v\n", err)
		} else {
			gt = groundtruth.Merge(existing, entries)
			fmt.Fprintf(os.Stderr, "Merged with %s (%d existing entries)\n", *mergePath, len(existing.Entries))
		}
	}

	data, err := yaml.Marshal(gt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling output: %v\n", err)
		os.Exit(1)
	}

	if *output == "" {
		os.Stdout.Write(data)
	} else {
		if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*output, data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", *output, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Written to %s (%d entries)\n", *output, len(gt.Entries))
	}
}

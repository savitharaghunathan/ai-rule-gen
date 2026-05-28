package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/groundtruth"
	"gopkg.in/yaml.v3"
)

func main() {
	logPath := flag.String("log", "", "Append structured output to this log file (overrides RULE_GEN_LOG)")
	agentFlag := flag.String("agent", "", "Name of the invoking agent (for log attribution)")
	modelFlag := flag.String("model", "", "LLM model powering the invoking agent (for log attribution)")
	oldArtifact := flag.String("old-artifact", "", "Old artifact coordinate (groupId:artifactId:version)")
	newArtifact := flag.String("new-artifact", "", "New artifact coordinate (groupId:artifactId:version)")
	fromGuide := flag.String("from-guide", "", "Extract ground truth from ingested guide markdown (bypasses japicmp)")
	output := flag.String("output", "", "Output path for ground_truth.yaml (default: stdout)")
	japicmpJar := flag.String("japicmp-jar", "", "Path to japicmp standalone JAR (auto-downloaded if not set)")
	mergePath := flag.String("merge", "", "Existing ground_truth.yaml to merge with")
	guideURL := flag.String("guide-url", "", "Migration guide URL for the output file")
	flag.Parse()

	cli.InitLog(*logPath, *agentFlag, *modelFlag)
	defer cli.CloseLog()

	if *fromGuide != "" && (*oldArtifact != "" || *newArtifact != "") {
		cli.Fail("invalid_arguments", "--from-guide is mutually exclusive with --old-artifact/--new-artifact", "ground-truth", "use either --from-guide or --old-artifact/--new-artifact", nil)
	}

	var entries []groundtruth.Entry

	if *fromGuide != "" {
		var err error
		entries, err = groundtruth.ExtractFromGuide(*fromGuide)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error extracting from guide: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Extracted %d FQNs from guide\n", len(entries))
	} else {
		if *oldArtifact == "" || *newArtifact == "" {
			cli.Fail("invalid_arguments", "--old-artifact and --new-artifact are required (or use --from-guide)", "ground-truth", "provide Maven coordinates as groupId:artifactId:version", nil)
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

		entries = groundtruth.ConvertChanges(changes)
	}

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

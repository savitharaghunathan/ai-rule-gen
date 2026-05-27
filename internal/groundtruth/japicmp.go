package groundtruth

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const defaultJapicmpVersion = "0.25.0"

// MavenCoord represents a Maven artifact coordinate.
type MavenCoord struct {
	GroupID    string
	ArtifactID string
	Version   string
}

// ParseCoord parses "groupId:artifactId:version" into a MavenCoord.
func ParseCoord(s string) (MavenCoord, error) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) != 3 {
		return MavenCoord{}, fmt.Errorf("invalid coordinate %q: expected groupId:artifactId:version", s)
	}
	return MavenCoord{GroupID: parts[0], ArtifactID: parts[1], Version: parts[2]}, nil
}

// MavenURL returns the Maven Central download URL for a JAR.
func (c MavenCoord) MavenURL() string {
	groupPath := strings.ReplaceAll(c.GroupID, ".", "/")
	return fmt.Sprintf("https://repo1.maven.org/maven2/%s/%s/%s/%s-%s.jar",
		groupPath, c.ArtifactID, c.Version, c.ArtifactID, c.Version)
}

// DownloadJAR downloads a JAR from Maven Central to the given directory.
// Returns the local path. Skips download if the file already exists.
func DownloadJAR(coord MavenCoord, dir string) (string, error) {
	filename := fmt.Sprintf("%s-%s.jar", coord.ArtifactID, coord.Version)
	dest := filepath.Join(dir, filename)

	if _, err := os.Stat(dest); err == nil {
		return dest, nil
	}

	resp, err := http.Get(coord.MavenURL())
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", coord.MavenURL(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading %s: HTTP %d", coord.MavenURL(), resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return "", fmt.Errorf("creating %s: %w", dest, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(dest)
		return "", fmt.Errorf("writing %s: %w", dest, err)
	}
	return dest, nil
}

// JapicmpJARURL returns the Maven Central URL for the japicmp standalone JAR.
func JapicmpJARURL(version string) string {
	return fmt.Sprintf(
		"https://repo1.maven.org/maven2/com/github/siom79/japicmp/japicmp/%s/japicmp-%s-jar-with-dependencies.jar",
		version, version)
}

// EnsureJapicmp downloads the japicmp standalone JAR if it doesn't exist at the given path.
func EnsureJapicmp(jarPath string) (string, error) {
	if jarPath != "" {
		if _, err := os.Stat(jarPath); err == nil {
			return jarPath, nil
		}
		return "", fmt.Errorf("japicmp JAR not found at %s", jarPath)
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	dest := filepath.Join(cacheDir, "ground-truth", fmt.Sprintf("japicmp-%s-jar-with-dependencies.jar", defaultJapicmpVersion))

	if _, err := os.Stat(dest); err == nil {
		return dest, nil
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", fmt.Errorf("creating cache dir: %w", err)
	}

	url := JapicmpJARURL(defaultJapicmpVersion)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("downloading japicmp: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading japicmp: HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(dest)
		return "", err
	}

	return dest, nil
}

// RunJapicmp runs japicmp and returns the XML diff output.
func RunJapicmp(japicmpJar, oldJar, newJar, outputXML string) error {
	cmd := exec.Command("java", "-jar", japicmpJar,
		"--old", oldJar,
		"--new", newJar,
		"--xml-file", outputXML,
		"--ignore-missing-classes",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("japicmp: %w\n%s", err, stderr.String())
	}
	return nil
}

// japicmpOutput represents the top-level XML output from japicmp.
type japicmpOutput struct {
	XMLName xml.Name       `xml:"japicmp"`
	Classes []japicmpClass `xml:"classes>class"`
}

type japicmpClass struct {
	FullyQualifiedName string           `xml:"fullyQualifiedName,attr"`
	ChangeStatus       string           `xml:"changeStatus,attr"`
	NewClass           string           `xml:"newClass,attr"`
	BinaryCompatible   string           `xml:"binaryCompatible,attr"`
	SourceCompatible   string           `xml:"sourceCompatible,attr"`
	Methods            []japicmpMethod  `xml:"methods>method"`
	Fields             []japicmpField   `xml:"fields>field"`
}

type japicmpMethod struct {
	Name         string `xml:"name,attr"`
	ChangeStatus string `xml:"changeStatus,attr"`
	NewName      string `xml:"newName,attr"`
	ReturnType   string `xml:"returnType,attr"`
}

type japicmpField struct {
	Name         string `xml:"name,attr"`
	ChangeStatus string `xml:"changeStatus,attr"`
}

// APIChange represents a single API change extracted from japicmp output.
type APIChange struct {
	OldAPI     string
	ChangeKind string // class_removed, class_modified, method_removed, method_changed, field_removed
	Detail     string
}

// ParseJapicmpXML parses japicmp XML output into a list of API changes.
func ParseJapicmpXML(path string) ([]APIChange, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading japicmp output: %w", err)
	}
	return parseJapicmpXMLData(data)
}

func parseJapicmpXMLData(data []byte) ([]APIChange, error) {
	var output japicmpOutput
	if err := xml.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("parsing japicmp XML: %w", err)
	}

	var changes []APIChange

	for _, cls := range output.Classes {
		switch cls.ChangeStatus {
		case "REMOVED":
			changes = append(changes, APIChange{
				OldAPI:     cls.FullyQualifiedName,
				ChangeKind: "class_removed",
				Detail:     "class removed",
			})
		case "MODIFIED":
			if cls.BinaryCompatible == "false" || cls.SourceCompatible == "false" {
				for _, m := range cls.Methods {
					if m.ChangeStatus == "REMOVED" {
						changes = append(changes, APIChange{
							OldAPI:     cls.FullyQualifiedName + "." + m.Name,
							ChangeKind: "method_removed",
							Detail:     "method removed",
						})
					} else if m.ChangeStatus == "MODIFIED" {
						changes = append(changes, APIChange{
							OldAPI:     cls.FullyQualifiedName + "." + m.Name,
							ChangeKind: "method_changed",
							Detail:     "method signature changed",
						})
					}
				}
				for _, f := range cls.Fields {
					if f.ChangeStatus == "REMOVED" {
						changes = append(changes, APIChange{
							OldAPI:     cls.FullyQualifiedName + "." + f.Name,
							ChangeKind: "field_removed",
							Detail:     "field removed",
						})
					}
				}
			}
		case "NEW":
			// New classes in a different package may indicate a move
			// but we can't determine the old→new mapping without additional context
		}
	}

	return changes, nil
}

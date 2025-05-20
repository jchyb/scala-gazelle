package maven

import (
	"encoding/json"
	"os"
	"fmt"
)

type ConfigFile struct {
	Dependencies  []Dependency
}

type Dependency struct {
	Coord              string   
	Packages           []string 
}

func loadConfiguration(filename string) (*ConfigFile, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var config ConfigFile
	var raw json.RawMessage
	if err := json.NewDecoder(f).Decode(&raw); err != nil {
		return nil, err
	}
	
	var vc versionedConfig
	err = json.Unmarshal(raw, &vc)
	
	if vc.Version == "2" {
		var c configFileV2
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, err
		}
		for name, artifact := range c.Artifacts {
			packages, exists := c.Packages[name]
			if !exists {
				packages = []string{}
			}
			dependency := Dependency{
				Coord: fmt.Sprintf("%s:%s", name, artifact.Version),
				Packages: packages, 
			}
			config.Dependencies = append(config.Dependencies, dependency)
		}
	} else {
		var c configFileV1
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, err
		}
		for _, depV1 := range c.DependencyTree.Dependencies {
			dependency := Dependency{
				Coord:    depV1.Coord,
				Packages: depV1.Packages,
			}
			config.Dependencies = append(config.Dependencies, dependency)
		}
}

	return &config, nil
}

// Version 0.1.0
type configFileV1 struct {
	DependencyTree dependencyTreeV1 `json:"dependency_tree,omitempty"`
}

type dependencyTreeV1 struct {
	ConflictResolution map[string]string `json:"conflict_resolution"`
	Dependencies       []depV1           `json:"dependencies"`
	Version            string            `json:"version"`
}

type depV1 struct {
	Coord              string   `json:"coord"`
	Dependencies       []string `json:"dependencies"`
	DirectDependencies []string `json:"directDependencies"`
	File               string   `json:"file"`
	MirrorUrls         []string `json:"mirror_urls,omitempty"`
	Packages           []string `json:"packages"`
	Sha256             string   `json:"sha256,omitempty"`
	URL                string   `json:"url,omitempty"`
	Exclusions         []string `json:"exclusions,omitempty"`
}

// Subtype of Version 2
type versionedConfig struct {
	Version string `json:"version"`
}

// Version 2
type configFileV2 struct {
	Artifacts    map[string]artifactV2          `json:"artifacts"`
	Dependencies map[string][]string            `json:"dependencies"`
	Packages     map[string][]string            `json:"packages"`
	Repositories map[string][]string            `json:"repositories"`
	Services     map[string]map[string][]string `json:"services"`
	Skipped      []string                       `json:"skipped"`
	Version      string                         `json:"version"`
}

type artifactV2 struct {
	ShaSums shaSumsV2 `json:"shasums"`
	Version string    `json:"version"`
}
type shaSumsV2 struct {
	Jar     string `json:"jar"` 
	Sources string `json:"sources"`
}



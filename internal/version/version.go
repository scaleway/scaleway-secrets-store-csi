package version

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

const minDriverVersion = "0.1.0"

// The following variables are meant to be set at build time from 'ldflags'
var (
	BuildDate    string
	BuildVersion string
	GoVersion    string
)

type Version struct {
	Version          string `json:"version"`          // Version of the binary
	BuildDate        string `json:"buildDate"`        // The date the binary was built
	GoVersion        string `json:"goVersion"`        // Version of Go the binary was built with
	MinDriverVersion string `json:"minDriverVersion"` // Minimum driver version the provider works with
}

func Get() *Version {
	return &Version{
		Version:          BuildVersion,
		BuildDate:        BuildDate,
		GoVersion:        GoVersion,
		MinDriverVersion: minDriverVersion,
	}
}

func (v *Version) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(*v)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (v *Version) MarshalYAML() ([]byte, error) {
	b, err := yaml.Marshal(*v)
	if err != nil {
		return nil, err
	}

	return b, nil
}

package config

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/scaleway/scaleway-sdk-go/scw"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Attributes  Attributes
	Credentials Credentials
	Secrets     []Secret
	TargetPath  string
	Permission  os.FileMode
}

type Credentials struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

type Attributes struct {
	APIURL                string `json:"apiURL"`
	Insecure              string `json:"insecure"`
	DefaultOrganizationID string `json:"defaultOrganizationID"`
	DefaultProjectID      string `json:"defaultProjectID"`
	DefaultRegion         string `json:"defaultRegion"`
	Objects               string `json:"objects"`
}

type Secret struct {
	ProjectID  string `yaml:"projectID"`
	ID         string `yaml:"secretID"`
	Path       string `yaml:"secretPath"`
	Name       string `yaml:"secretName"`
	Revision   string `yaml:"revision"`
	TargetPath string `yaml:"targetPath"`
}

func Parse(attributesStr, credentialsStr, targetPath, permissionStr string) (*Config, error) {
	config := &Config{
		TargetPath: targetPath,
	}

	var err error

	if err = json.Unmarshal([]byte(permissionStr), &config.Permission); err != nil {
		return nil, fmt.Errorf("failed to unmarshal permission: %w", err)
	}

	config.Attributes, err = parseAttributes(attributesStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse attributes: %w", err)
	}

	config.Credentials, err = parseCredentials(credentialsStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	config.Secrets, err = parseSecrets(config.Attributes.Objects, config.Attributes.DefaultOrganizationID, config.Attributes.DefaultProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse secrets: %w", err)
	}

	if err = config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return config, nil
}

func parseAttributes(attributesStr string) (Attributes, error) {
	attributes := defaultAttributes()

	if err := json.Unmarshal([]byte(attributesStr), &attributes); err != nil {
		return Attributes{}, fmt.Errorf("failed to unmarshal attributes: %w", err)
	}

	return attributes, nil
}

func parseCredentials(credentialsStr string) (Credentials, error) {
	credentials := defaultCredentials()

	if err := json.Unmarshal([]byte(credentialsStr), &credentials); err != nil {
		return Credentials{}, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return credentials, nil
}

func parseSecrets(secretsStr string, defaultOrganizationID, defaultProjectID string) ([]Secret, error) {
	secrets := make([]Secret, 0)

	if err := yaml.Unmarshal([]byte(secretsStr), &secrets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secrets: %w", err)
	}

	// Sanitize secrets
	for i := range secrets {
		if secrets[i].ID == "" {
			// TargetPath is optional and defaults to ./secret_path/secret_name
			if secrets[i].TargetPath == "" {
				secrets[i].TargetPath = filepath.Join(secrets[i].Path, secrets[i].Name)
				secrets[i].TargetPath = strings.TrimPrefix(secrets[i].TargetPath, "/")
			}

			// ProjectID is optional and defaults to default project ID or default organization ID
			secrets[i].ProjectID = cmp.Or(secrets[i].ProjectID, defaultProjectID, defaultOrganizationID)
		}

		// Revision is optional and defaults to 'latest_enabled'
		if secrets[i].Revision == "" {
			secrets[i].Revision = "latest_enabled"
		}

		if secrets[i].Path != "" {
			secrets[i].Path = filepath.Clean(secrets[i].Path)
		}

		if secrets[i].TargetPath != "" {
			secrets[i].TargetPath = filepath.Clean(secrets[i].TargetPath)
		}
	}

	return secrets, nil
}

func (c *Config) Validate() error {
	if c.TargetPath == "" {
		return errors.New("missing target path")
	}

	if err := c.Attributes.Validate(); err != nil {
		return fmt.Errorf("invalid attributes: %w", err)
	}

	if err := c.Credentials.Validate(); err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}

	if len(c.Secrets) == 0 {
		return errors.New("invalid secrets: no secret specified")
	}

	for i, secret := range c.Secrets {
		if err := secret.Validate(); err != nil {
			return fmt.Errorf("invalid secret %d: %w", i, err)
		}
	}

	return nil
}

func (a Attributes) Validate() error {
	if a.APIURL == "" {
		return errors.New("missing api url")
	}

	if a.DefaultRegion == "" {
		return errors.New("missing default region")
	}

	if region := scw.Region(a.DefaultRegion); !region.Exists() {
		return errors.New("region does not exist")
	}

	return nil
}

func (c Credentials) Validate() error {
	if c.AccessKey == "" {
		return errors.New("missing access key")
	}

	if c.SecretKey == "" {
		return errors.New("missing secret key")
	}

	return nil
}

func (s Secret) Validate() error {
	if s.ID != "" && (s.Path != "" || s.Name != "") {
		return errors.New("secret can be specified by id or path but not both")
	}

	if s.Revision == "" {
		return errors.New("missing revision")
	}

	if s.TargetPath == "" {
		return errors.New("missing target path")
	}

	// TargetPath must be a relative path
	if filepath.IsAbs(s.TargetPath) {
		return errors.New("target path must be relative")
	}

	// Secret is specified by path
	if s.ID == "" {
		if s.Path == "" || s.Name == "" {
			return errors.New("missing secret path or secret name")
		}

		if s.ProjectID == "" {
			return errors.New("missing project id")
		}

		// SecretPath must be an absolute path
		if filepath.IsLocal(s.Path) {
			return errors.New("secret path must be absolute")
		}
	}

	return nil
}

func defaultAttributes() Attributes {
	attributes := Attributes{
		APIURL:   "https://api.scaleway.com",
		Insecure: "false",
	}

	profile := scw.LoadEnvProfile()

	if profile.APIURL != nil {
		attributes.APIURL = *profile.APIURL
	}

	if profile.Insecure != nil && *profile.Insecure {
		attributes.Insecure = "true"
	}

	if profile.DefaultOrganizationID != nil {
		attributes.DefaultOrganizationID = *profile.DefaultOrganizationID
	}

	if profile.DefaultProjectID != nil {
		attributes.DefaultProjectID = *profile.DefaultProjectID
	}

	if profile.DefaultRegion != nil {
		attributes.DefaultRegion = *profile.DefaultRegion
	}

	return attributes
}

func defaultCredentials() Credentials {
	credentials := Credentials{}

	profile := scw.LoadEnvProfile()

	if profile.AccessKey != nil {
		credentials.AccessKey = *profile.AccessKey
	}

	if profile.SecretKey != nil {
		credentials.SecretKey = *profile.SecretKey
	}

	return credentials
}

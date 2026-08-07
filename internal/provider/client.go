package provider

import (
	"fmt"

	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/scaleway-secrets-store-csi/internal/config"
	"github.com/scaleway/scaleway-secrets-store-csi/internal/version"
)

const (
	userAgentPrefix = "secrets-store-csi-driver-provider-scw"
)

func newScalewayClient(cfg *config.Config) (*scw.Client, error) {
	userAgent := fmt.Sprintf("%s/%s", userAgentPrefix, version.BuildVersion)

	clientOpts := []scw.ClientOption{
		scw.WithAPIURL(cfg.Attributes.APIURL),
		scw.WithAuth(cfg.Credentials.AccessKey, cfg.Credentials.SecretKey),
		scw.WithUserAgent(userAgent),
	}

	if cfg.Attributes.Insecure == "true" {
		clientOpts = append(clientOpts, scw.WithInsecure())
	}

	if organizationID := cfg.Attributes.DefaultOrganizationID; organizationID != "" {
		clientOpts = append(clientOpts, scw.WithDefaultOrganizationID(organizationID))
	}

	if projectID := cfg.Attributes.DefaultProjectID; projectID != "" {
		clientOpts = append(clientOpts, scw.WithDefaultProjectID(projectID))
	}

	if region := scw.Region(cfg.Attributes.DefaultRegion); region != "" {
		clientOpts = append(clientOpts, scw.WithDefaultRegion(region))
	}

	client, err := scw.NewClient(clientOpts...)
	if err != nil {
		return nil, err
	}

	return client, nil
}

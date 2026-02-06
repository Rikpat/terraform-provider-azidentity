// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ provider.Provider = &AzIdentityProvider{}
var _ provider.ProviderWithEphemeralResources = &AzIdentityProvider{}

// AzIdentityProvider defines the provider implementation.
type AzIdentityProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// AzIdentityProviderModel describes the provider data model.
type AzurePipelinesCredentialModel struct {
	TenantID            types.String `tfsdk:"tenant_id"`
	ClientID            types.String `tfsdk:"client_id"`
	ServiceConnectionID types.String `tfsdk:"service_connection_id"`
	SystemAccessToken   types.String `tfsdk:"system_access_token"`
}

type ClientSecretCredentialModel struct {
	TenantID     types.String `tfsdk:"tenant_id"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

type ClientCertificateCredentialModel struct {
	TenantID            types.String `tfsdk:"tenant_id"`
	ClientID            types.String `tfsdk:"client_id"`
	CertificatePath     types.String `tfsdk:"certificate_path"`
	CertificatePassword types.String `tfsdk:"certificate_password"`
}

type ManagedIdentityCredentialModel struct {
	ClientID types.String `tfsdk:"client_id"`
}

type AzureCLICredentialModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type WorkloadIdentityCredentialModel struct {
	TenantID types.String `tfsdk:"tenant_id"`
	ClientID types.String `tfsdk:"client_id"`
}

// AzIdentityProviderModel describes the provider data model.
type AzIdentityProviderModel struct {
	Cloud                       types.String                      `tfsdk:"cloud"`
	Credentials                 types.Set                         `tfsdk:"credentials"`
	AzurePipelinesCredential    *AzurePipelinesCredentialModel    `tfsdk:"azure_pipelines_credential"`
	ClientSecretCredential      *ClientSecretCredentialModel      `tfsdk:"client_secret_credential"`
	ClientCertificateCredential *ClientCertificateCredentialModel `tfsdk:"client_certificate_credential"`
	ManagedIdentityCredential   *ManagedIdentityCredentialModel   `tfsdk:"managed_identity_credential"`
	AzureCLICredential          *AzureCLICredentialModel          `tfsdk:"azure_cli_credential"`
	WorkloadIdentityCredential  *WorkloadIdentityCredentialModel  `tfsdk:"workload_identity_credential"`
}

func (p *AzIdentityProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "azidentity"
	resp.Version = p.version
}

// Provider configuration is primarily about selecting and configuring credential sources.
func (p *AzIdentityProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Provider used for authenticating with resources supporting EntraID authentication.

Main usage is generating a token using Azure Pipelines Workload Federation Identity in IaC pipelines and falling back to azure_cli for local testing, but supports more credential types.

Most credentials have options like selecting client_id and tenant_id, except for *environment* and *azure_cli* credentials which take all the options from external sources.
		`,
		Attributes: map[string]schema.Attribute{
			"cloud": schema.StringAttribute{
				MarkdownDescription: "Cloud environment to target. Possible values are: ***AzurePublic*** (default), *AzureGovernment*, *AzureChina*",
				Optional:            true,
			},
			"credentials": schema.SetAttribute{
				ElementType: types.StringType,

				MarkdownDescription: `List of credentials to try. They will be tried in the specified order. 
	
	Supported types are (enabled by default in this order are in bold, similar to defaultAzureCredential): 
	- **environment**
	- **azure_pipelines** 
	- **workload_identity**
	- **managed_identity**
	- **azure_cli**
	- client_secret
	- client_certificate`,
				Optional: true,
			},
			"azure_pipelines_credential": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration block for Azure Pipelines Credential. If using TerraformTask@5, no configuration needed unless you want to use different service connection than used for terraform. If using AzureCLI@2 or AzurePowershell@5, you need to also set SYSTEM_ACCESSTOKEN env variable, or provide access token as terraform variable.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"tenant_id": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Optional tenant_id if it's different from used service connection (*ARM_TENANT_ID* or *AZURE_TENANT_ID*)",
					},
					"client_id": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Optional client_id if it's different from used service connection (*ARM_CLIENT_ID* or *AZURE_CLIENT_ID*)",
					},
					"service_connection_id": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Optional Azure DevOps Service Connection ID, if it's different from used service connection (*ARM_OIDC_AZURE_SERVICE_CONNECTION_ID* or *AZURESUBSCRIPTION_SERVICE_CONNECTION_ID*)",
					},
					"system_access_token": schema.StringAttribute{
						Optional:            true,
						Sensitive:           true,
						MarkdownDescription: "Optional OIDC request token, if not using Terraform@5 task, or not setting *SYSTEM_ACCESSTOKEN* env variable",
					},
				},
			},
			"workload_identity_credential": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for workload identity credential. You can provide custom client_id and tenant_id if using multiple workload identities on single pod.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"tenant_id": schema.StringAttribute{Optional: true},
					"client_id": schema.StringAttribute{Optional: true},
				},
			},
			"managed_identity_credential": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for Managed Identity credential (optional `client_id` for user-assigned identity).",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"client_id": schema.StringAttribute{Optional: true},
				},
			},
			"client_secret_credential": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for a client secret credential. All properties are required, as there's already environment_credential that provides same functionality with env variables.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"tenant_id":     schema.StringAttribute{Required: true},
					"client_id":     schema.StringAttribute{Required: true},
					"client_secret": schema.StringAttribute{Required: true, Sensitive: true},
				},
			},
			"client_certificate_credential": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for a client certificate credential. All properties are required, as there's already environment_credential that provides same functionality with env variables.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"tenant_id":            schema.StringAttribute{Required: true},
					"client_id":            schema.StringAttribute{Required: true},
					"certificate_path":     schema.StringAttribute{Required: true},
					"certificate_password": schema.StringAttribute{Required: true, Sensitive: true},
				},
			},
		},
	}
}

func (p *AzIdentityProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring provider")
	// Read all env vars

	var data AzIdentityProviderModel

	if resp.Diagnostics.Append(req.Config.Get(ctx, &data)...); resp.Diagnostics.HasError() {
		return
	}

	cred, diags := setupCredentialChain(ctx, &data)

	if resp.Diagnostics.Append(diags...); resp.Diagnostics.HasError() {
		return
	}

	resp.EphemeralResourceData = cred
}

func (p *AzIdentityProvider) Resources(ctx context.Context) []func() resource.Resource {
	return nil
}

func (p *AzIdentityProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		NewTokenEphemeralResource,
	}
}

func (p *AzIdentityProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return nil
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AzIdentityProvider{
			version: version,
		}
	}
}

// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Select cloud configuration based on the input string, display warning to user if it's not recognized.
func selectCloud(c string) (cloud.Configuration, diag.Diagnostic) {
	switch c {
	case "AzureChina":
		return cloud.AzureChina, nil
	case "AzureGovernment":
		return cloud.AzureGovernment, nil
	case "", "AzurePublic":
		return cloud.AzurePublic, nil
	}
	return cloud.AzurePublic, diag.NewAttributeWarningDiagnostic(path.Root("cloud"), "Invalid cloud value", fmt.Sprintf("The provided cloud value '%s' is not recognized. Falling back to AzurePublic.", c))
}

func selectCredentials(in *[]types.String, data *AzIdentityProviderModel, clientOptions azcore.ClientOptions) ([]azcore.TokenCredential, diag.Diagnostics) {
	out := make([]azcore.TokenCredential, len(*in))
	diags := diag.Diagnostics{}
	for i, credential := range *in {
		var err error = nil
		c := credential.ValueString()
		switch c {
		case "environment":
			out[i], err = azidentity.NewEnvironmentCredential(
				&azidentity.EnvironmentCredentialOptions{
					ClientOptions: clientOptions,
				},
			)
		case "managedIdentity":
			var id azidentity.ClientID
			if stringId := data.ManagedIdentityCredential.ClientID.ValueString(); stringId != "" {
				id = azidentity.ClientID(stringId)
			}
			out[i], err = azidentity.NewManagedIdentityCredential(
				&azidentity.ManagedIdentityCredentialOptions{
					ClientOptions: clientOptions,
					ID:            id,
				})
		case "azureCLI":
			out[i], err = azidentity.NewAzureCLICredential(nil)
		case "workloadIdentity":
			out[i], err = azidentity.NewWorkloadIdentityCredential(
				// Defaults solved by the SDK (AZURE_CLIENT_ID, AZURE_TENANT_ID)
				&azidentity.WorkloadIdentityCredentialOptions{
					ClientOptions: clientOptions,
					ClientID:      data.WorkloadIdentityCredential.ClientID.ValueString(),
					TenantID:      data.WorkloadIdentityCredential.TenantID.ValueString(),
				})
		case "azurePipelines":
			ok := false
			clientID := data.AzurePipelinesCredential.ClientID.ValueString()
			if clientID == "" {
				if clientID, ok = os.LookupEnv("ARM_CLIENT_ID"); !ok {
					if clientID, ok = os.LookupEnv("AZURE_CLIENT_ID"); !ok {
						diags.AddAttributeWarning(path.Root("azure_pipelines_credential"), "Missing Client ID, skipping credential", "Client ID is not set. Please provide a value or set the environment variable ARM_CLIENT_ID or AZURE_CLIENT_ID.")
					}
				}
			}
			tenantID := data.AzurePipelinesCredential.TenantID.ValueString()
			if tenantID == "" {
				if tenantID, ok = os.LookupEnv("ARM_TENANT_ID"); !ok {
					if tenantID, ok = os.LookupEnv("AZURE_TENANT_ID"); !ok {
						diags.AddAttributeWarning(path.Root("azure_pipelines_credential"), "Missing Tenant ID, skipping credential", "Tenant ID is not set. Please provide a value or set the environment variable ARM_TENANT_ID or AZURE_TENANT_ID.")
					}
				}
			}
			serviceConnectionID := data.AzurePipelinesCredential.ServiceConnectionID.ValueString()
			if serviceConnectionID == "" {
				if serviceConnectionID, ok = os.LookupEnv("ARM_OIDC_AZURE_SERVICE_CONNECTION_ID"); !ok {
					if serviceConnectionID, ok = os.LookupEnv("AZURESUBSCRIPTION_SERVICE_CONNECTION_ID"); !ok {
						diags.AddAttributeWarning(path.Root("azure_pipelines_credential"), "Missing Service Connection ID, skipping credential", "Service Connection ID is not set. Please provide a value or set the environment variable ARM_OIDC_AZURE_SERVICE_CONNECTION_ID or AZURESUBSCRIPTION_SERVICE_CONNECTION_ID.")
					}
				}
			}
			systemAccessToken := data.AzurePipelinesCredential.ServiceConnectionID.ValueString()
			if serviceConnectionID == "" {
				if serviceConnectionID, ok = os.LookupEnv("ARM_OIDC_REQUEST_TOKEN"); !ok {
					if serviceConnectionID, ok = os.LookupEnv("SYSTEM_ACCESSTOKEN"); !ok {
						diags.AddAttributeWarning(path.Root("azure_pipelines_credential"), "Missing Service Connection ID, skipping credential", "Service Connection ID is not set. Please provide a value or set the environment variable ARM_OIDC_AZURE_SERVICE_CONNECTION_ID or AZURESUBSCRIPTION_SERVICE_CONNECTION_ID.")
					}
				}
			}
			out[i], err = azidentity.NewAzurePipelinesCredential(
				tenantID,
				clientID,
				serviceConnectionID,
				systemAccessToken,
				&azidentity.AzurePipelinesCredentialOptions{
					ClientOptions: clientOptions,
				},
			)
		case "clientSecret":
			// No defaults, if user needs to use env variables, they can use environment provider
			out[i], err = azidentity.NewClientSecretCredential(
				data.ClientSecretCredential.TenantID.ValueString(),
				data.ClientSecretCredential.ClientID.ValueString(),
				data.ClientSecretCredential.ClientSecret.ValueString(),
				&azidentity.ClientSecretCredentialOptions{
					ClientOptions: clientOptions,
				},
			)
		case "clientCertificate":
			certData, err := os.ReadFile(data.ClientCertificateCredential.CertificatePath.ValueString())
			if err != nil {
				diags.AddAttributeWarning(path.Root("client_certificate_credential"), "Failed to read certificate file", err.Error())
				break
			}
			cert, key, err := azidentity.ParseCertificates(certData, []byte(data.ClientCertificateCredential.CertificatePassword.ValueString()))
			if err != nil {
				diags.AddAttributeWarning(path.Root("client_certificate_credential"), "Failed to parse certificate file", err.Error())
				break
			}
			// No defaults, if user needs to use env variables, they can use environment provider
			out[i], err = azidentity.NewClientCertificateCredential(
				data.ClientCertificateCredential.TenantID.ValueString(),
				data.ClientCertificateCredential.ClientID.ValueString(),
				cert,
				key,
				&azidentity.ClientCertificateCredentialOptions{
					ClientOptions: clientOptions,
				},
			)
		default:
			diags.AddAttributeWarning(path.Root("credentials").AtSetValue(credential), "Invalid Credential type", fmt.Sprintf("Unknown type '%s'. Credential ignored. Check if you accidentally misspelled the credential type.", c))
		}
		if err != nil {
			diags.AddAttributeWarning(path.Root("credentials").AtSetValue(credential), fmt.Sprintf("Error setting up credential '%s'.", c), err.Error())
		}
	}
	return out, diags
}

func setupCredentialChain(ctx context.Context, data *AzIdentityProviderModel) (*azidentity.ChainedTokenCredential, diag.Diagnostics) {
	// Get credential types to use
	credentialTypes := make([]types.String, 0, len(data.Credentials.Elements()))
	diags := data.Credentials.ElementsAs(ctx, &credentialTypes, false)

	// Get cloud type
	cloud, diag := selectCloud(data.Cloud.ValueString())
	diags.Append(diag)

	credentials, newDiags := selectCredentials(&credentialTypes, data, azcore.ClientOptions{Cloud: cloud})
	diags.Append(newDiags...)

	cred, err := azidentity.NewChainedTokenCredential(credentials, nil)
	if err != nil {
		diags.AddError("Failed setting up credential chain", err.Error())
	}
	return cred, diags
}

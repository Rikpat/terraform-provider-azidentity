# Copyright (c) HashiCorp, Inc.

provider "azidentity" {
  # Simple pipeline identity with cli fallback for local
  credentials = ["azure_pipelines", "azure_cli"]
}

provider "azidentity" {
  alias       = "client2"
  credentials = ["azure_pipelines", "workload_identity"]
  # If running in pipeline agent, but use different service connection as azurerm
  azure_pipelines_credential = {
    service_connection_id = "6174d1c2-f44f-410c-93f1-82a19d400eb8"
    client_id             = "db64e57b-7500-4ece-b682-e8fa8c20d9d5"
    tenant_id             = "6aafbe4c-9457-415e-b57d-834fe4d09c7d"
  }
  # Same identity, if running in AKS cluster
  workload_identity_credential = {
    client_id = "db64e57b-7500-4ece-b682-e8fa8c20d9d5"
    tenant_id = "6aafbe4c-9457-415e-b57d-834fe4d09c7d"
  }
}
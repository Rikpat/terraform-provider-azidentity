# Copyright (c) HashiCorp, Inc.

ephemeral "azidentity_token" "token" {
  scopes = ["https://management.azure.com/.default"]
}

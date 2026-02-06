// Copyright (c) HashiCorp, Inc.

package validator

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	// Should work for all, but only string needed
	// _ validator.Bool    = ValueBasedValidator{}
	// _ validator.Float32 = ValueBasedValidator{}
	// _ validator.Float64 = ValueBasedValidator{}
	// _ validator.Int32   = ValueBasedValidator{}
	// _ validator.Int64   = ValueBasedValidator{}
	// _ validator.Number  = ValueBasedValidator{}
	_ validator.String = ValueBasedValidator{}
)

type ValueBasedValidator struct {
	ElementValidators map[string]validator.String
}

// Description returns a plain text description of the validator's behavior, suitable for a practitioner to understand its impact.
func (v ValueBasedValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

// MarkdownDescription returns a markdown formatted description of the validator's behavior, suitable for a practitioner to understand its impact.
func (v ValueBasedValidator) MarkdownDescription(ctx context.Context) string {
	return "Uses validators for specific values"
}

// Validate runs the main validation logic of the validator, reading configuration data out of `req` and updating `resp` with diagnostics.
func (v ValueBasedValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// If the value is unknown or null, there is nothing to validate.
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	if elementValidator, ok := v.ElementValidators[req.ConfigValue.ValueString()]; ok {
		elementValidator.ValidateString(ctx, req, resp)
	}
}

func ValueBased(validators map[string]validator.String) ValueBasedValidator {
	return ValueBasedValidator{ElementValidators: validators}
}

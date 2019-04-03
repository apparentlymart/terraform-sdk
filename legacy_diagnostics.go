package tfsdk

func diagnosticsFromWarnsAndErrs(warns []string, errs []error) Diagnostics {
	if len(warns) == 0 && len(errs) == 0 {
		return nil
	}
	diags := make(Diagnostics, 0, len(warns)+len(errs))

	for _, warn := range warns {
		diags = diags.Append(Diagnostic{
			Severity: Warning,
			Summary:  warn,
		})
	}
	for _, err := range errs {
		diags = diags.Append(Diagnostic{
			Severity: Error,
			Summary:  FormatError(err),
		})
	}

	return diags
}

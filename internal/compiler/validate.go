package compiler

// Validate checks one .north document with the same parser used by code generation.
func Validate(source []byte) error {
	prepared, _, err := compileClientComponent("Document", source)
	if err != nil {
		return err
	}
	_, err = parseComponent("Document", string(prepared))
	return err
}

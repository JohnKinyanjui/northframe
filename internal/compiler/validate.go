package compiler

// Validate checks one .north document with the same parser used by code generation.
func Validate(source []byte) error {
	options := clientCompileOptions{}
	if block, _, found, err := extractPropsBlock(string(source)); err != nil {
		return err
	} else if found {
		props, _, err := parseProps(block)
		if err != nil {
			return err
		}
		options.Props = props
	}
	prepared, _, err := compileClientComponentWithOptions("Document", source, options)
	if err != nil {
		return err
	}
	_, err = parseComponent("Document", string(prepared))
	return err
}

package parser

import (
	"fmt"
	"strings"
)

type debugField struct {
	key   string
	value string
}

func debugStruct(name string, fields []debugField) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("%s{\n", name))

	for _, field := range fields {
		builder.WriteString(fmt.Sprintf("    %s: ", field.key))
		builder.WriteString(strings.TrimSpace(indent(field.value)))
		builder.WriteString(",\n")
	}

	builder.WriteString("}")
	return builder.String()
}

func debugSlice(values []string) string {
	builder := strings.Builder{}
	builder.WriteString("[]{\n")

	for _, value := range values {
		builder.WriteString(indent(value))
		builder.WriteString(",\n")
	}

	builder.WriteString("}")
	return builder.String()
}

func indent(value string) string {
	builder := strings.Builder{}

	for index, value := range strings.Split(value, "\n") {
		if index == 0 {
			builder.WriteString(fmt.Sprintf("    %s", value))
		} else {
			builder.WriteString(fmt.Sprintf("\n    %s", value))
		}
	}

	return builder.String()
}

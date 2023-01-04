package logger

import (
	"fmt"
	"strings"
)

type DebugField struct {
	Key   string
	Value string
}

func DebugStruct(name string, fields []DebugField) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("%s{\n", name))

	for _, field := range fields {
		builder.WriteString(fmt.Sprintf("    %s: ", field.Key))
		builder.WriteString(strings.TrimSpace(indent(field.Value)))
		builder.WriteString(",\n")
	}

	builder.WriteString("}")
	return builder.String()
}

func DebugSlice(values []string) string {
	builder := strings.Builder{}
	builder.WriteString("[]{\n")

	for _, value := range values {
		builder.WriteString(indent(value))
		builder.WriteString(",\n")
	}

	builder.WriteString("}")
	return builder.String()
}

func DebugMap(pairs map[string]string) string {
	builder := strings.Builder{}
	builder.WriteString("map[]{\n")

	for key, value := range pairs {
		builder.WriteString(fmt.Sprintf("    %s: ", key))
		builder.WriteString(strings.TrimSpace(indent(value)))
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

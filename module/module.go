package module

import (
	"bytes"
	_ "embed"
	"io"
	"strconv"
	"strings"
	"text/template"
)

var (
	//go:embed readme.tmpl
	readme string

	readmeTemplate = template.Must(template.New("").Parse(readme)) //nolint:gochecknoglobals
)

const whitespace = " \t\n"

func ToSoftpackYML(name string, contents string) io.Reader {
	var (
		inHelp               bool
		packages             = []string{""}
		description, version string
	)

	for _, line := range strings.Split(contents, "\n") {
		line = strings.Trim(line, whitespace)

		if inHelp {
			if line == "}" {
				inHelp = false
			} else if strings.HasPrefix(line, "puts stderr") {
				line, _ = strconv.Unquote(strings.ReplaceAll(strings.Trim(strings.TrimPrefix(line, "puts stderr"), whitespace), "\\$", "$")) //nolint:errcheck

				description += "  " + line + "\n"
			}
		} else if strings.HasPrefix(line, "proc ModulesHelp") {
			inHelp = true
		} else if strings.HasPrefix(line, "module-whatis ") {
			line, _ = strconv.Unquote(strings.Trim(strings.TrimPrefix(line, "module-whatis"), whitespace)) //nolint:errcheck
			line = strings.Trim(line, whitespace)

			if strings.HasPrefix(line, "Name:") {
				if line = strings.TrimPrefix(line, "Name:"); line != "" {
					parts := strings.Split(line, ":")
					nameParts := strings.FieldsFunc(parts[0], func(r rune) bool {
						switch r {
						case ' ', '\n', '\t':
							return true
						}

						return false
					})

					if nameParts[0] != "" {
						name = nameParts[0]
					}

					if len(parts) > 1 {
						version = strings.Trim(parts[1], whitespace)
					}
				}
			} else if strings.HasPrefix(line, "Version:") {
				vers := strings.Fields(strings.TrimPrefix(line, "Version:"))
				if len(vers) > 0 && vers[0] != "" {
					version = vers[0]
				}
			} else if strings.HasPrefix(line, "Packages:") {
				for _, pkg := range strings.FieldsFunc(strings.Trim(strings.TrimPrefix(line, "Packages:"), whitespace), func(r rune) bool {
					switch r {
					case ' ', '\t', ',', '\n':
						return true
					}

					return false
				}) {
					if pkg != "" {
						packages = append(packages, pkg)
					}
				}
			}
		}
	}

	if version != "" {
		name += "@" + version
	}

	packages[0] = name

	return strings.NewReader("description: |\n" + description + "packages:\n  - " + strings.Join(packages, "\n  - ") + "\n")
}

func GenerateEnvReadme(modulePath string) io.Reader {
	var buf bytes.Buffer

	readmeTemplate.Execute(&buf, modulePath) //nolint:errcheck

	return &buf
}

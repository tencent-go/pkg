package tsdoc

import (
	"bytes"
	"fmt"
	"path"
	"strings"
	"text/template"
	"unicode"

	"github.com/tencent-go/pkg/doc/schema"
	"github.com/tencent-go/pkg/types"
	"github.com/tencent-go/pkg/util"
	"github.com/sirupsen/logrus"
)

func NewSchemaFiles(schemaPackages []*schema.Package, parentDir ...string) []util.DataFile {
	var schemaTmp = `
declare namespace {{.Namespace}} {
{{range .ClassItems}}
    interface {{.Name}} {
{{.Properties}};
    }
{{end}}{{range .EnumItems}}
{{if .Description}}    // {{.Description}}{{end}}
    type {{.Name}} = {{.Values}};
{{end}}
}
`

	t, err := template.New("typescript_schema").Parse(schemaTmp)
	if err != nil {
		logrus.Fatalf("parse schema template failed: %v", err)
		return nil
	}
	var res []util.DataFile
	dir := path.Join(parentDir...)
	for _, pkg := range schemaPackages {
		tmpData := schemaTmpData{
			Namespace: pkg.Name,
		}
		if pkg.Name == "" {
			logrus.Fatalf("package name is empty")
		}
		for _, c := range pkg.Classes {
			item := classItem{
				Name: c.Name,
			}
			var ps []string
			for _, f := range c.Fields {
				// 校驗屬性名稱是否合法
				if len(f.Name) == 0 {
					logrus.WithField("class", c).Fatalf("field name is empty")
				}
				if unicode.IsUpper(rune(f.Name[0])) {
					logrus.Warnf("field name should not start with uppercase: %s ,class: %s,package: %s", f.Name, c.Name, c.Package.Name)
				}
				separator := ":"
				if f.Optional {
					separator = "?:"
				}
				tp := parseType(f.Type, pkg)
				name := f.Name
				if strings.Contains(name, "-") || strings.Contains(name, "_") {
					name = fmt.Sprintf("'%s'", name)
				}
				str := fmt.Sprintf("        %s%s %s", name, separator, tp)
				ps = append(ps, str)
			}
			item.Properties = strings.Join(ps, ";\n")
			tmpData.ClassItems = append(tmpData.ClassItems, item)
		}
		for _, e := range pkg.Enums {
			item := enumItem{
				Name: e.Name,
			}
			var descriptions []string
			if len(e.Items) == 0 {
				item.Values = "any"
			} else {
				var values []string
				for _, it := range e.Items {
					if e.IsNumeric {
						values = append(values, fmt.Sprintf("%d", it.Value))
					} else {
						values = append(values, fmt.Sprintf("'%s'", it.Value))
					}
					if len(it.LocalizedInfo) != 0 {
						info := it.LocalizedInfo.Get(types.DefaultLocale)
						if label := info.Label; label != "" {
							description := fmt.Sprintf("%v: %s", it.Value, label)
							if info.Tip != "" {
								description += fmt.Sprintf(" (%s)", info.Tip)
							}
							descriptions = append(descriptions, description)
						}
					}
				}
				item.Values = strings.Join(values, " | ")
				item.Description = strings.Join(descriptions, ", ")
			}
			tmpData.EnumItems = append(tmpData.EnumItems, item)
		}
		var buf bytes.Buffer
		if err = t.Execute(&buf, tmpData); err != nil {
			logrus.Fatalf("execute schema template failed: %v", err)
		}
		res = append(res, util.DataFile{
			Dir:  dir,
			Name: pkg.Name + ".d.ts",
			Data: buf.Bytes(),
		})
	}
	return res
}

type schemaTmpData struct {
	Namespace  string
	ClassItems []classItem
	EnumItems  []enumItem
}

type classItem struct {
	Name       string
	Properties string
}

type enumItem struct {
	Name        string
	Values      string
	Description string
}

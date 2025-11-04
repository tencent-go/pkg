package env

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/tencent-go/pkg/validation"

	"github.com/tencent-go/pkg/types"
)

type ConfigReader[T any] interface {
	Read() T
}

type ReaderBuilder[T any] interface {
	WithPrefix(prefix string) ReaderBuilder[T]
	WithAllFieldsRequired(required bool) ReaderBuilder[T]
	Build() ConfigReader[T]
}

func NewReaderBuilder[T any]() ReaderBuilder[T] {
	return &readerBuilder[T]{}
}

type readerBuilder[T any] struct {
	prefix            string
	allFieldsRequired *bool
	reader            *configReader[T]
}

func (c *readerBuilder[T]) WithPrefix(prefix string) ReaderBuilder[T] {
	return &readerBuilder[T]{
		prefix:            prefix,
		allFieldsRequired: c.allFieldsRequired,
	}
}

func (c *readerBuilder[T]) WithAllFieldsRequired(required bool) ReaderBuilder[T] {
	return &readerBuilder[T]{
		prefix:            c.prefix,
		allFieldsRequired: &required,
	}
}

func (c *readerBuilder[T]) Build() ConfigReader[T] {
	if c.reader != nil {
		return c.reader
	}
	reader := &configReader[T]{
		prefix:            c.prefix,
		allFieldsRequired: c.allFieldsRequired,
		typ:               reflect.TypeOf(new(T)).Elem(),
	}
	c.reader = reader
	configs = append(configs, reader)
	return reader
}

type printable interface {
	printState()
	parse()
}

var configs []printable

func PrintState() {
	for _, c := range configs {
		c.parse()
		c.printState()
	}
}

type configReader[T any] struct {
	prefix            string
	allFieldsRequired *bool
	parsed            *T
	once              sync.Once
	fields            []fieldInfo
	hasErr            bool
	typ               reflect.Type
}

func (c *configReader[T]) Get() T {
	if c.parsed == nil {
		c.parse()
	}
	if c.hasErr {
		c.printState()
		panic("Configuration parsing error")
	}
	return *c.parsed
}

func (c *configReader[T]) Read() T {
	if c.parsed == nil {
		c.parse()
	}
	if c.hasErr {
		c.printState()
		panic("Configuration parsing error")
	}
	return *c.parsed
}

func (c *configReader[T]) printState() {
	var reportBuilder strings.Builder
	withPrefix := ""
	if c.prefix != "" {
		withPrefix = fmt.Sprintf(" with prefix %s", c.prefix)
	}
	reportBuilder.WriteString(fmt.Sprintf("\nStruct [%s]%s environment variable state:\n", c.typ.String(), withPrefix))

	// 计算每列的最大宽度
	columnWidths := c.calculateColumnWidths()

	// 构建格式字符串
	format := fmt.Sprintf("| %%-%ds | %%-%ds | %%-%ds | %%-%ds | %%-%ds | %%-%ds | %%-%ds |\n",
		columnWidths.key, columnWidths.kind, columnWidths.currentValue,
		columnWidths.required, columnWidths.example, columnWidths.description, columnWidths.issue)

	// 计算总宽度
	totalWidth := columnWidths.key + columnWidths.kind + columnWidths.currentValue +
		columnWidths.required + columnWidths.example + columnWidths.description + columnWidths.issue + 20 // 7列 * 2空格 + 6分隔符 + 2外框

	horizontalLine := "+" + strings.Repeat("-", totalWidth) + "+\n"
	reportBuilder.WriteString(horizontalLine)
	reportBuilder.WriteString(fmt.Sprintf(format,
		"Key",
		"Type",
		"Current Value",
		"Required",
		"Example",
		"Description",
		"Issue"),
	)
	reportBuilder.WriteString(horizontalLine)

	for _, info := range c.fields {
		currentValue := info.value
		if currentValue == "" {
			currentValue = info.defaultValue
		}
		if currentValue != "" && info.defaultValue == currentValue {
			currentValue = fmt.Sprintf("%s (default)", currentValue)
		}
		required := ""
		if !info.omitempty {
			required = "yes"
		}
		reportBuilder.WriteString(fmt.Sprintf(format,
			c.truncateString(info.key, columnWidths.key),
			c.truncateString(info.kind, columnWidths.kind),
			c.truncateString(currentValue, columnWidths.currentValue),
			c.truncateString(required, columnWidths.required),
			c.truncateString(info.example, columnWidths.example),
			c.truncateString(info.description, columnWidths.description),
			c.truncateString(info.errorMessage, columnWidths.issue),
		))
	}
	reportBuilder.WriteString(horizontalLine)
	fmt.Print(reportBuilder.String())
}

type columnWidths struct {
	key          int
	kind         int
	currentValue int
	required     int
	example      int
	description  int
	issue        int
}

func (c *configReader[T]) calculateColumnWidths() columnWidths {
	// 最小宽度
	minWidths := columnWidths{
		key:          8,  // "Key" 的最小宽度
		kind:         8,  // "Type" 的最小宽度
		currentValue: 12, // "Current Value" 的最小宽度
		required:     8,  // "Required" 的最小宽度
		example:      8,  // "Example" 的最小宽度
		description:  11, // "Description" 的最小宽度
		issue:        5,  // "Issue" 的最小宽度
	}

	// 最大宽度
	maxWidths := columnWidths{
		key:          30,
		kind:         20,
		currentValue: 50,
		required:     10,
		example:      40,
		description:  60,
		issue:        30,
	}

	// 计算实际需要的宽度
	widths := columnWidths{
		key:          max(minWidths.key, min(maxWidths.key, len("Key"))),
		kind:         max(minWidths.kind, min(maxWidths.kind, len("Type"))),
		currentValue: max(minWidths.currentValue, min(maxWidths.currentValue, len("Current Value"))),
		required:     max(minWidths.required, min(maxWidths.required, len("Required"))),
		example:      max(minWidths.example, min(maxWidths.example, len("Example"))),
		description:  max(minWidths.description, min(maxWidths.description, len("Description"))),
		issue:        max(minWidths.issue, min(maxWidths.issue, len("Issue"))),
	}

	// 根据实际数据调整宽度
	for _, info := range c.fields {
		currentValue := info.value
		if currentValue == "" {
			currentValue = info.defaultValue
		}
		if currentValue != "" && info.defaultValue == currentValue {
			currentValue = fmt.Sprintf("%s (default)", currentValue)
		}
		required := ""
		if !info.omitempty {
			required = "yes"
		}

		widths.key = max(widths.key, min(maxWidths.key, len(info.key)))
		widths.kind = max(widths.kind, min(maxWidths.kind, len(info.kind)))
		widths.currentValue = max(widths.currentValue, min(maxWidths.currentValue, len(currentValue)))
		widths.required = max(widths.required, min(maxWidths.required, len(required)))
		widths.example = max(widths.example, min(maxWidths.example, len(info.example)))
		widths.description = max(widths.description, min(maxWidths.description, len(info.description)))
		widths.issue = max(widths.issue, min(maxWidths.issue, len(info.errorMessage)))
	}

	return widths
}

func (c *configReader[T]) truncateString(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return "..."
	}
	return s[:maxWidth-3] + "..."
}

type fieldInfo struct {
	key          string
	omitempty    bool
	defaultValue string
	example      string
	description  string
	value        string
	kind         string
	errorMessage string
}

func setFieldValue(value string, field reflect.Value, omitempty bool) error {
	if value == "" {
		if omitempty {
			return nil
		}
		return fmt.Errorf("required value is empty")
	}
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(value, 10, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid integer value: %v", err)
		}
		field.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(value, 10, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value: %v", err)
		}
		field.SetUint(i)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(value, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid float value: %v", err)
		}
		field.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %v", err)
		}
		field.SetBool(b)
	case reflect.Slice:
		values := strings.Split(value, ",")
		slice := reflect.MakeSlice(field.Type(), len(values), len(values))
		for i, v := range values {
			v = strings.TrimSpace(v)
			elem := slice.Index(i)
			if err := setFieldValue(v, elem, false); err != nil {
				return err
			}
		}
		field.Set(slice)
	default:
		return fmt.Errorf("unsupported type: %v", field.Type())
	}
	return nil
}

func (c *configReader[T]) collectFieldInfo(val reflect.Value) []fieldInfo {
	typ := val.Type()
	var fields []fieldInfo

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if !fieldType.IsExported() {
			continue
		}

		envTag := fieldType.Tag.Get("env")
		if envTag != "" && envTag[0] == '-' {
			continue
		}
		envTag = strings.TrimSuffix(envTag, ",omitempty")
		if envTag == "" {
			envTag = toSnakeCase(fieldType.Name)
		}
		if c.prefix != "" {
			envTag = c.prefix + "_" + envTag
		}

		if fieldType.Anonymous {
			if field.Kind() == reflect.Ptr {
				if field.IsNil() {
					field.Set(reflect.New(field.Type().Elem()))
				}
				field = field.Elem()
			}
			if field.Kind() == reflect.Struct {
				fields = append(fields, c.collectFieldInfo(field)...)
			}
			continue
		}

		validator, _, validatorErr := validation.CreateStructFieldConfig(fieldType, true, "env")

		omitempty := true

		if validator != nil {
			omitempty = !validator.IsRequired()
		}

		if c.allFieldsRequired != nil {
			omitempty = !*c.allFieldsRequired
		}

		for field.Kind() == reflect.Ptr {
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			field = field.Elem()
		}

		value := os.Getenv(envTag)

		info := fieldInfo{
			key:          envTag,
			omitempty:    omitempty,
			defaultValue: fieldType.Tag.Get("default"),
			example:      fieldType.Tag.Get("example"),
			description:  fieldType.Tag.Get("description"),
			value:        value,
			kind:         field.Type().String(),
		}

		if field.Type().Implements(enumInterface) {
			enum := field.Interface().(types.IEnum).Enum()
			if info.example == "" {
				var items []string
				for _, item := range enum.Items() {
					items = append(items, fmt.Sprintf("%v", item.Value))
				}
				info.example = strings.Join(items, ", ")
			}
		}
		if validatorErr != nil {
			info.errorMessage = validatorErr.Error()
			fields = append(fields, info)
			continue
		}

		if value == "" && info.defaultValue != "" {
			value = info.defaultValue
		}

		if err := setFieldValue(value, field, omitempty); err != nil {
			info.errorMessage = err.Error()
		} else {
			if value != "" {
				if validator != nil {
					err = validator.Validate(field)
					if err != nil {
						info.errorMessage = err.Error()
					}
				}
			}
		}
		fields = append(fields, info)
	}
	return fields
}

var enumInterface = reflect.TypeOf((*types.IEnum)(nil)).Elem()

func (c *configReader[T]) parse() {
	c.once.Do(func() {
		t := new(T)
		val := reflect.ValueOf(t).Elem()
		c.fields = c.collectFieldInfo(val)
		hasError := false
		for _, info := range c.fields {
			if info.errorMessage != "" {
				hasError = true
				break
			}
		}
		c.hasErr = hasError
		c.parsed = t
	})
}

// toSnakeCase 将驼峰命名转换为下划线命名
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToUpper(r))
	}
	return result.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

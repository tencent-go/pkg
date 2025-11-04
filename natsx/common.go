package natsx

import (
	"regexp"
	"strings"

	"github.com/tencent-go/pkg/ctxx"
	"github.com/tencent-go/pkg/util"
	"github.com/nats-io/nats.go"
)

type MsgIdGetter interface {
	MsgID() string
}

func newNatsHeader(ctx ctxx.Context) nats.Header {
	header := nats.Header{}
	header.Set("traceId", ctx.GetTraceID().String())
	header.Set("operator", ctx.GetOperator())
	header.Set("caller", ctx.GetCaller())
	header.Set("locale", string(ctx.GetLocale()))
	return header
}

// replaceSubjectPlaceholders 替换subject中的占位符
// subject: 包含占位符的主题字符串，如 "user.{userId}.created"
// args: 占位符对应的值，按顺序提供
// 返回: 替换后的字符串和未找到值的占位符列表
func replaceSubjectPlaceholders(subject string, args ...string) (result string, missingPlaceholders []string) {
	// 匹配 {placeholder} 格式的正则表达式
	placeholderRegex := util.PlaceholderRegex

	// 记录已处理的占位符数量
	processedCount := 0

	// 一次遍历完成替换
	result = placeholderRegex.ReplaceAllStringFunc(subject, func(match string) string {
		// 提取占位符名称
		placeholder := match[1 : len(match)-1] // 去掉 { 和 }

		// 根据处理顺序获取对应的args值
		if processedCount < len(args) && args[processedCount] != "" {
			processedCount++
			return args[processedCount-1]
		} else {
			missingPlaceholders = append(missingPlaceholders, placeholder)
			processedCount++
			return "*"
		}
	})

	return result, missingPlaceholders
}

var nameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

func isSubjectsContains(outers []string, inner string) bool {
	for _, outer := range outers {
		if isSubjectContains(outer, inner) {
			return true
		}
	}
	return false
}

func isSubjectContains(outer, inner string) bool {
	oParts := strings.Split(outer, ".")
	iParts := strings.Split(inner, ".")
	for i, oPart := range oParts {
		if oPart == ">" {
			return true
		}
		if oPart == "*" {
			continue
		}
		if i >= len(iParts) {
			return false
		}
		iPart := iParts[i]
		if oPart == iPart {
			continue
		}
		if strings.HasPrefix(iPart, "{") && strings.HasSuffix(iPart, "}") {
			continue
		}
		return false
	}
	return len(oParts) <= len(iParts)
}

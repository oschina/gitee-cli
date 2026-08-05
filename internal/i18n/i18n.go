package i18n

import (
	"fmt"

	"gitee.com/oschina/gitee-cli/internal/config"
)

func T(key string) string {
	locale := config.Locale()
	if locale == "zh_CN" {
		if v, ok := zhCN[key]; ok {
			return v
		}
	}
	if v, ok := en[key]; ok {
		return v
	}
	return key
}

func Tf(key string, args ...any) string {
	return fmt.Sprintf(T(key), args...)
}

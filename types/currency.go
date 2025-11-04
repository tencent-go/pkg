package types

type Currency string

func (c Currency) Enum() Enum {
	return RegisterEnum(USD, EUR, JPY, GBP, CNY, INR, BRL, RUB, CAD, AUD, ZAR, MXN, SGD, NZD, CHF, HKD, PHP)
}

const (
	USD Currency = "USD" // 美元
	EUR Currency = "EUR" // 欧元
	JPY Currency = "JPY" // 日元
	GBP Currency = "GBP" // 英镑
	CNY Currency = "CNY" // 人民币
	INR Currency = "INR" // 印度卢比
	BRL Currency = "BRL" // 巴西雷亚尔
	RUB Currency = "RUB" // 俄罗斯卢布
	CAD Currency = "CAD" // 加拿大元
	AUD Currency = "AUD" // 澳大利亚元
	ZAR Currency = "ZAR" // 南非兰特
	MXN Currency = "MXN" // 墨西哥比索
	SGD Currency = "SGD" // 新加坡元
	NZD Currency = "NZD" // 新西兰元
	CHF Currency = "CHF" // 瑞士法郎
	HKD Currency = "HKD" // 港元
	PHP Currency = "PHP" // 菲律宾比索
)

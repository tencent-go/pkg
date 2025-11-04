package types

type CountryCode string

const (
	CountryCodeHongKong           CountryCode = "852"
	CountryCodeUnitedStates       CountryCode = "1"
	CountryCodeCanada             CountryCode = "1"
	CountryCodeUnitedKingdom      CountryCode = "44"
	CountryCodeAustralia          CountryCode = "61"
	CountryCodeIndia              CountryCode = "91"
	CountryCodeGermany            CountryCode = "49"
	CountryCodeFrance             CountryCode = "33"
	CountryCodeJapan              CountryCode = "81"
	CountryCodeSouthKorea         CountryCode = "82"
	CountryCodeSingapore          CountryCode = "65"
	CountryCodeThailand           CountryCode = "66"
	CountryCodeVietnam            CountryCode = "84"
	CountryCodeIndonesia          CountryCode = "62"
	CountryCodeMalaysia           CountryCode = "60"
	CountryCodeItaly              CountryCode = "39"
	CountryCodeRussia             CountryCode = "7"
	CountryCodeBrazil             CountryCode = "55"
	CountryCodeSouthAfrica        CountryCode = "27"
	CountryCodeMexico             CountryCode = "52"
	CountryCodeArgentina          CountryCode = "54"
	CountryCodeChile              CountryCode = "56"
	CountryCodeColombia           CountryCode = "57"
	CountryCodeSaudiArabia        CountryCode = "966"
	CountryCodeUnitedArabEmirates CountryCode = "971"
	CountryCodeEgypt              CountryCode = "20"
	CountryCodeTurkey             CountryCode = "90"
	CountryCodeSpain              CountryCode = "34"
	CountryCodePortugal           CountryCode = "351"
	CountryCodeNetherlands        CountryCode = "31"
	CountryCodeChina              CountryCode = "86"
	CountryCodeSwitzerland        CountryCode = "41"
	CountryCodeSweden             CountryCode = "46"
	CountryCodeDenmark            CountryCode = "45"
	CountryCodeNorway             CountryCode = "47"
	CountryCodeNewZealand         CountryCode = "64"
	CountryCodePakistan           CountryCode = "92"
	CountryCodeBangladesh         CountryCode = "880"
	CountryCodeSriLanka           CountryCode = "94"
	CountryCodeNepal              CountryCode = "977"
	CountryCodeMyanmar            CountryCode = "95"
	CountryCodePhilippines        CountryCode = "63"
	CountryCodeIsrael             CountryCode = "972"
)

func (c CountryCode) Enum() Enum {
	return RegisterEnum(CountryCodeHongKong, CountryCodeUnitedStates, CountryCodeCanada, CountryCodeUnitedKingdom, CountryCodeAustralia, CountryCodeIndia, CountryCodeGermany, CountryCodeFrance, CountryCodeJapan, CountryCodeSouthKorea, CountryCodeSingapore, CountryCodeThailand, CountryCodeVietnam, CountryCodeIndonesia, CountryCodeMalaysia, CountryCodeItaly, CountryCodeRussia, CountryCodeBrazil, CountryCodeSouthAfrica, CountryCodeMexico, CountryCodeArgentina, CountryCodeChile, CountryCodeColombia, CountryCodeSaudiArabia, CountryCodeUnitedArabEmirates, CountryCodeEgypt, CountryCodeTurkey, CountryCodeSpain, CountryCodePortugal, CountryCodeNetherlands, CountryCodeChina, CountryCodeSwitzerland, CountryCodeSweden, CountryCodeDenmark, CountryCodeNorway, CountryCodeNewZealand, CountryCodePakistan, CountryCodeBangladesh, CountryCodeSriLanka, CountryCodeNepal, CountryCodeMyanmar, CountryCodePhilippines, CountryCodeIsrael)
}

func init() {
	ExtendEnum(CountryCodeHongKong).Label("Hong Kong").
		Value(CountryCodeUnitedStates).Label("United States").
		Value(CountryCodeCanada).Label("Canada").
		Value(CountryCodeUnitedKingdom).Label("United Kingdom").
		Value(CountryCodeAustralia).Label("Australia").
		Value(CountryCodeIndia).Label("India").
		Value(CountryCodeGermany).Label("Germany").
		Value(CountryCodeFrance).Label("France").
		Value(CountryCodeJapan).Label("Japan").
		Value(CountryCodeSouthKorea).Label("South Korea").
		Value(CountryCodeSingapore).Label("Singapore").
		Value(CountryCodeThailand).Label("Thailand").
		Value(CountryCodeVietnam).Label("Vietnam").
		Value(CountryCodeIndonesia).Label("Indonesia").
		Value(CountryCodeMalaysia).Label("Malaysia").
		Value(CountryCodeItaly).Label("Italy").
		Value(CountryCodeRussia).Label("Russia").
		Value(CountryCodeBrazil).Label("Brazil").
		Value(CountryCodeSouthAfrica).Label("South Africa").
		Value(CountryCodeMexico).Label("Mexico").
		Value(CountryCodeArgentina).Label("Argentina").
		Value(CountryCodeChile).Label("Chile").
		Value(CountryCodeColombia).Label("Colombia").
		Value(CountryCodeSaudiArabia).Label("Saudi Arabia").
		Value(CountryCodeUnitedArabEmirates).Label("United Arab Emirates").
		Value(CountryCodeEgypt).Label("Egypt").
		Value(CountryCodeTurkey).Label("Turkey").
		Value(CountryCodeSpain).Label("Spain").
		Value(CountryCodePortugal).Label("Portugal").
		Value(CountryCodeNetherlands).Label("Netherlands").
		Value(CountryCodeChina).Label("China").
		Value(CountryCodeSwitzerland).Label("Switzerland").
		Value(CountryCodeSweden).Label("Sweden").
		Value(CountryCodeDenmark).Label("Denmark").
		Value(CountryCodeNorway).Label("Norway").
		Value(CountryCodeNewZealand).Label("New Zealand").
		Value(CountryCodePakistan).Label("Pakistan").
		Value(CountryCodeBangladesh).Label("Bangladesh").
		Value(CountryCodeSriLanka).Label("Sri Lanka").
		Value(CountryCodeNepal).Label("Nepal").
		Locale(ZhCN).
		Value(CountryCodeUnitedStates).Label("美国").
		Value(CountryCodeCanada).Label("加拿大").
		Value(CountryCodeUnitedKingdom).Label("英国").
		Value(CountryCodeAustralia).Label("澳大利亚").
		Value(CountryCodeIndia).Label("印度").
		Value(CountryCodeGermany).Label("德国").
		Value(CountryCodeFrance).Label("法国").
		Value(CountryCodeJapan).Label("日本").
		Value(CountryCodeSouthKorea).Label("韩国").
		Value(CountryCodeSingapore).Label("新加坡").
		Value(CountryCodeThailand).Label("泰国").
		Value(CountryCodeVietnam).Label("越南").
		Value(CountryCodeIndonesia).Label("印度尼西亚").
		Value(CountryCodeMalaysia).Label("马来西亚").
		Value(CountryCodeItaly).Label("意大利").
		Value(CountryCodeRussia).Label("俄罗斯").
		Value(CountryCodeBrazil).Label("巴西").
		Value(CountryCodeSouthAfrica).Label("南非").
		Value(CountryCodeMexico).Label("墨西哥").
		Value(CountryCodeArgentina).Label("阿根廷").
		Value(CountryCodeChile).Label("智利").
		Value(CountryCodeColombia).Label("哥伦比亚").
		Value(CountryCodeSaudiArabia).Label("沙特阿拉伯").
		Value(CountryCodeUnitedArabEmirates).Label("阿联酋").
		Value(CountryCodeEgypt).Label("埃及").
		Value(CountryCodeTurkey).Label("土耳其").
		Value(CountryCodeSpain).Label("西班牙").
		Value(CountryCodePortugal).Label("葡萄牙").
		Value(CountryCodeNetherlands).Label("荷兰").
		Value(CountryCodeChina).Label("中国").
		Value(CountryCodeSwitzerland).Label("瑞士").
		Value(CountryCodeSweden).Label("瑞典").
		Value(CountryCodeDenmark).Label("丹麦").
		Value(CountryCodeNorway).Label("挪威").
		Value(CountryCodeNewZealand).Label("新西兰").
		Value(CountryCodePakistan).Label("巴基斯坦").
		Value(CountryCodeBangladesh).Label("孟加拉").
		Value(CountryCodeSriLanka).Label("斯里兰卡").
		Value(CountryCodeNepal).Label("尼泊尔")
}

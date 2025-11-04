package types

import (
	"github.com/tencent-go/pkg/errx"
)

// BCP 47
type Locale string

const (
	// DefaultLocale 為預設值，通常表示無特定地區設定
	DefaultLocale Locale = ""

	// 中文地區設定
	ZhCN Locale = "zh-CN" // 中國大陸，簡體中文
	ZhTW Locale = "zh-TW" // 台灣，繁體中文
	ZhHK Locale = "zh-HK" // 香港，繁體中文

	// 英語地區設定
	EnUS Locale = "en-US" // 美國英語
	EnGB Locale = "en-GB" // 英國英語
	EnAU Locale = "en-AU" // 澳大利亞英語
	EnCA Locale = "en-CA" // 加拿大英語
	EnIN Locale = "en-IN" // 印度英語

	// 法語地區設定
	FrFR Locale = "fr-FR" // 法國法語
	FrCA Locale = "fr-CA" // 加拿大法語

	// 德語地區設定
	DeDE Locale = "de-DE" // 德國德語
	DeCH Locale = "de-CH" // 瑞士德語

	// 西班牙語地區設定
	EsES Locale = "es-ES" // 西班牙西班牙語
	EsMX Locale = "es-MX" // 墨西哥西班牙語
	EsUS Locale = "es-US" // 美國西班牙語

	// 日語地區設定
	JaJP Locale = "ja-JP" // 日本日語

	// 韓語地區設定
	KoKR Locale = "ko-KR" // 韓國韓語

	// 俄語地區設定
	RuRU Locale = "ru-RU" // 俄羅斯俄語

	// 葡萄牙語地區設定
	PtBR Locale = "pt-BR" // 巴西葡萄牙語
	PtPT Locale = "pt-PT" // 葡萄牙葡萄牙語

	// 阿拉伯語地區設定
	ArSA Locale = "ar-SA" // 沙烏地阿拉伯阿拉伯語
	ArEG Locale = "ar-EG" // 埃及阿拉伯語

	// 印地語地區設定
	HiIN Locale = "hi-IN" // 印度印地語

	// 義大利語地區設定
	ItIT Locale = "it-IT" // 義大利義大利語
	ItCH Locale = "it-CH" // 瑞士義大利語

	// 荷蘭語地區設定
	NlNL Locale = "nl-NL" // 荷蘭荷蘭語
	NlBE Locale = "nl-BE" // 比利時荷蘭語

	// 波蘭語地區設定
	PlPL Locale = "pl-PL" // 波蘭波蘭語

	// 越南語地區設定
	ViVN Locale = "vi-VN" // 越南越南語

	// 泰語地區設定
	ThTH Locale = "th-TH" // 泰國泰語

	// 希臘語地區設定
	ElGR Locale = "el-GR" // 希臘希臘語

	// 土耳其語地區設定
	TrTR Locale = "tr-TR" // 土耳其土耳其語

	// 瑞典語地區設定
	SvSE Locale = "sv-SE" // 瑞典瑞典語

	FilPH Locale = "fil-PH" // 菲律宾语（菲律宾）
)

func (l Locale) Enum() Enum {
	return RegisterEnum(DefaultLocale, ZhCN, ZhTW, ZhHK, EnUS, EnGB, EnAU, EnCA, EnIN, FrFR, FrCA, DeDE, FilPH, DeCH, EsES, EsMX, EsUS, JaJP, KoKR, RuRU, PtBR, PtPT, ArSA, ArEG, HiIN, ItIT, ItCH, NlNL, NlBE, PlPL, ViVN, ThTH, ElGR, TrTR, SvSE)
}

type LocalizedValues[T any] map[Locale]T

func (v LocalizedValues[T]) Get(l Locale) T {
	val, ok := v[l]
	if !ok {
		return v[DefaultLocale]
	}
	return val
}

func (v LocalizedValues[T]) Validate() errx.Error {
	for l := range v {
		if !l.Enum().Contains(l) {
			return errx.Newf("invalid locale: %s", l)
		}
	}
	return nil
}

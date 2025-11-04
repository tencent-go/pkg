package types

type Gender string

const (
	GenderUnKnow Gender = "un_know"
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
)

func (g Gender) Enum() Enum {
	return RegisterEnum(GenderUnKnow, GenderMale, GenderFemale)
}

func init() {
	ExtendEnum(GenderUnKnow).Label("UnKnow").
		Value(GenderMale).Label("Male").
		Value(GenderFemale).Label("Female").
		Locale(ZhCN).
		Value(GenderUnKnow).Label("未知").
		Value(GenderMale).Label("男性").
		Value(GenderFemale).Label("女性")
}

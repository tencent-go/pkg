package types

type Platform string

const (
	PlatformPC      Platform = "pc"
	PlatformAndroid Platform = "android"
	PlatformIOS     Platform = "ios"
)

func (p Platform) Enum() Enum {
	return RegisterEnum(PlatformPC, PlatformAndroid, PlatformIOS)
}

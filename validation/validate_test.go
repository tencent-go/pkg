package validation

import (
	"reflect"
	"testing"
	"time"

	"github.com/tencent-go/pkg/errx"
	"github.com/stretchr/testify/assert"
)

type User struct {
	Name     string    `validate:"required,min=2,max=50" json:"name"`
	Age      int       `validate:"required,min=0,max=150" json:"age"`
	Email    string    `validate:"required,pattern=^[a-zA-Z0-9._%+\\-]+@[a-zA-Z0-9.\\-]+\\.[a-zA-Z]{2,}$" json:"email"`
	Birthday time.Time `validate:"required" json:"birthday"`
}

type Address struct {
	Street string `validate:"required" json:"street"`
	City   string `validate:"required" json:"city"`
}

type UserWithAddress struct {
	Name    string  `validate:"required" json:"name"`
	Address Address `validate:"dive" json:"address"`
}

type CustomString string

type ValidatableString string

func (v ValidatableString) Validate() errx.Error {
	str := string(v)
	if str == "" {
		return errx.Validation.WithMsg("value cannot be empty").Err()
	}
	if len(str) < 2 {
		return errx.Validation.WithMsg("value length must be greater than 2").Err()
	}
	return nil
}

func customStringValidatorBuilder(typ reflect.Type, rule *Rule) (Validator, bool) {
	if typ != reflect.TypeOf(CustomString("")) {
		return nil, false
	}
	return func(value reflect.Value) errx.Error {
		str := value.String()
		if str == "" {
			return errx.Validation.WithMsg("value cannot be empty").Err()
		}
		if len(str) < 2 {
			return errx.Validation.WithMsg("value length must be greater than 2").Err()
		}
		return nil
	}, true
}

func init() {
	RegisterValidatorBuilder(customStringValidatorBuilder)
}

func TestValidateStructWithCache(t *testing.T) {
	now := time.Now()

	t.Run("基本驗證-成功", func(t *testing.T) {
		u := &User{
			Name:     "張三",
			Age:      25,
			Email:    "zhangsan@example.com",
			Birthday: now,
		}
		err := ValidateStructWithCache(u, WithLabelTag("json"))
		assert.NoError(t, err)
	})

	t.Run("基本驗證-失敗", func(t *testing.T) {
		u := &User{
			Name:     "",
			Age:      -1,
			Email:    "not-an-email",
			Birthday: time.Time{},
		}
		err := ValidateStructWithCache(u, WithLabelTag("json"))
		assert.Error(t, err)
	})

	t.Run("嵌套結構體-成功", func(t *testing.T) {
		u := &UserWithAddress{
			Name: "張三",
			Address: Address{
				Street: "人民路",
				City:   "北京",
			},
		}
		err := ValidateStructWithCache(u, WithLabelTag("json"))
		assert.NoError(t, err)
	})

	t.Run("嵌套結構體-失敗", func(t *testing.T) {
		u := &UserWithAddress{
			Name:    "張三",
			Address: Address{},
		}
		err := ValidateStructWithCache(u, WithLabelTag("json"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "street")
	})

	t.Run("Map驗證-成功", func(t *testing.T) {
		type Cfg struct {
			Settings map[string]string `validate:"dive,keys,required,endkeys,required" json:"settings"`
		}
		cfg := &Cfg{
			Settings: map[string]string{"key1": "value1", "key2": "value2"},
		}
		err := ValidateStructWithCache(cfg, WithLabelTag("json"))
		assert.NoError(t, err)
	})

	t.Run("Map驗證-失敗", func(t *testing.T) {
		type Cfg struct {
			Settings map[string]string `validate:"dive,keys,required,endkeys,required" json:"settings"`
		}
		cfg := &Cfg{Settings: map[string]string{"key1": "", "key2": "value2"}}
		err := ValidateStructWithCache(cfg, WithLabelTag("json"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key1")
	})

	t.Run("自定義驗證器-成功", func(t *testing.T) {
		type Req struct {
			Name CustomString `validate:"required" json:"name"`
		}
		r := &Req{Name: CustomString("張三")}
		err := ValidateStructWithCache(r, WithLabelTag("json"))
		assert.NoError(t, err)
	})

	t.Run("自定義驗證器-失敗", func(t *testing.T) {
		type Req struct {
			Name CustomString `validate:"required" json:"name"`
		}
		r := &Req{Name: CustomString("")}
		err := ValidateStructWithCache(r, WithLabelTag("json"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("Validatable接口-成功", func(t *testing.T) {
		type Req struct {
			Name ValidatableString `validate:"required" json:"name"`
		}
		r := &Req{Name: ValidatableString("張三")}
		err := ValidateStructWithCache(r, WithLabelTag("json"))
		assert.NoError(t, err)
	})

	t.Run("Validatable接口-失敗", func(t *testing.T) {
		type Req struct {
			Name ValidatableString `validate:"required" json:"name"`
		}
		r := &Req{Name: ValidatableString("")}
		err := ValidateStructWithCache(r, WithLabelTag("json"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("正則表達式驗證-成功", func(t *testing.T) {
		type RegexTest struct {
			Phone string `validate:"required,pattern=^1[3-9]\\d{9}$" json:"phone"`
		}
		v := &RegexTest{Phone: "13812345678"}
		err := ValidateStructWithCache(v, WithLabelTag("json"))
		assert.NoError(t, err)
	})

	t.Run("正則表達式驗證-失敗", func(t *testing.T) {
		type RegexTest struct {
			Phone string `validate:"required,pattern=^1[3-9]\\d{9}$" json:"phone"`
		}
		v := &RegexTest{Phone: "12345678"}
		err := ValidateStructWithCache(v, WithLabelTag("json"))
		assert.Error(t, err)
	})

	t.Run("枚舉值驗證-成功", func(t *testing.T) {
		type EnumTest struct {
			Status string `validate:"required,pattern=^(active|inactive|pending)$" json:"status"`
		}
		v := &EnumTest{Status: "active"}
		err := ValidateStructWithCache(v, WithLabelTag("json"))
		assert.NoError(t, err)
	})

	t.Run("枚舉值驗證-失敗", func(t *testing.T) {
		type EnumTest struct {
			Status string `validate:"required,pattern=^(active|inactive|pending)$" json:"status"`
		}
		v := &EnumTest{Status: "invalid"}
		err := ValidateStructWithCache(v, WithLabelTag("json"))
		assert.Error(t, err)
	})
}

func TestAdditionalValidationScenarios(t *testing.T) {
	t.Run("指针类型验证-成功", func(t *testing.T) {
		type PointerTest struct {
			Name *string `validate:"required" json:"name"`
		}
		name := "张三"
		v := &PointerTest{Name: &name}
		err := ValidateStructWithCache(v, WithLabelTag("json"))
		assert.NoError(t, err)
	})

	t.Run("指针类型验证-失败", func(t *testing.T) {
		type PointerTest struct {
			Name *string `validate:"required" json:"name"`
		}
		v := &PointerTest{Name: nil}
		err := ValidateStructWithCache(v, WithLabelTag("json"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("切片类型验证-成功", func(t *testing.T) {
		type SliceTest struct {
			Tags []string `validate:"dive,required,min=2" json:"tags"`
		}
		v := &SliceTest{Tags: []string{"tag1", "tag2"}}
		err := ValidateStructWithCache(v, WithLabelTag("json"))
		assert.NoError(t, err)
	})

	t.Run("切片类型验证-失败", func(t *testing.T) {
		type SliceTest struct {
			Tags []string `validate:"dive,required,min=2" json:"tags"`
		}
		v := &SliceTest{Tags: []string{"t", "tag2"}}
		err := ValidateStructWithCache(v, WithLabelTag("json"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tags")
	})

	t.Run("忽略字段验证", func(t *testing.T) {
		type IgnoreTest struct {
			Name  string `validate:"required" json:"name"`
			Email string `validate:"ignore" json:"email"`
		}
		v := &IgnoreTest{Name: "张三", Email: ""}
		err := ValidateStructWithCache(v, WithLabelTag("json"))
		assert.NoError(t, err)
	})

	t.Run("零值验证-成功", func(t *testing.T) {
		type ZeroTest struct {
			Age int `validate:"min=0" json:"age"`
		}
		v := &ZeroTest{Age: 0}
		err := ValidateStructWithCache(v, WithLabelTag("json"))
		assert.NoError(t, err)
	})

	t.Run("数字类型min/max验证-成功", func(t *testing.T) {
		type NumberTest struct {
			IntNum   int     `validate:"min=1,max=100" json:"int_num"`
			FloatNum float64 `validate:"min=1.5,max=100.5" json:"float_num"`
		}
		v := &NumberTest{
			IntNum:   50,
			FloatNum: 50.5,
		}
		err := ValidateStructWithCache(v, WithLabelTag("json"))
		assert.NoError(t, err)
	})

	t.Run("数字类型min/max验证-失败", func(t *testing.T) {
		type NumberTest struct {
			IntNum   int     `validate:"required,min=1,max=100" json:"int_num"`
			FloatNum float64 `validate:"required,min=1.5,max=100.5" json:"float_num"`
		}
		v := &NumberTest{
			IntNum:   0,
			FloatNum: 0.5,
		}
		err := ValidateStructWithCache(v, WithLabelTag("json"))
		assert.Error(t, err)
	})

	t.Run("时间类型验证-成功", func(t *testing.T) {
		type TimeTest struct {
			StartTime time.Time `validate:"required" json:"start_time"`
			EndTime   time.Time `validate:"required" json:"end_time"`
		}
		now := time.Now()
		v := &TimeTest{
			StartTime: now,
			EndTime:   now.Add(24 * time.Hour),
		}
		err := ValidateStructWithCache(v, WithLabelTag("json"))
		assert.NoError(t, err)
	})

	t.Run("时间类型验证-失败", func(t *testing.T) {
		type TimeTest struct {
			StartTime time.Time `validate:"required" json:"start_time"`
			EndTime   time.Time `validate:"required" json:"end_time"`
		}
		v := &TimeTest{
			StartTime: time.Time{},
			EndTime:   time.Time{},
		}
		err := ValidateStructWithCache(v, WithLabelTag("json"))
		assert.Error(t, err)
		assert.True(t, err.Error() == "Field 'start_time': value is required" ||
			err.Error() == "Field 'end_time': value is required")
	})

	t.Run("多标签组合验证-成功", func(t *testing.T) {
		type MultiTagTest struct {
			Phone string `validate:"required,pattern=^1[3-9]\\d{9}$" json:"phone"`
			Age   int    `validate:"required,min=18,max=100" json:"age"`
			Email string `validate:"required,pattern=^[a-zA-Z0-9._%+\\-]+@[a-zA-Z0-9.\\-]+\\.[a-zA-Z]{2,}$,required" json:"email"`
		}
		v := &MultiTagTest{
			Phone: "13812345678",
			Age:   25,
			Email: "test@example.com",
		}
		err := ValidateStructWithCache(v, WithLabelTag("json"))
		assert.NoError(t, err)
	})

	t.Run("多标签组合验证-失败", func(t *testing.T) {
		type MultiTagTest struct {
			Phone string `validate:"required,pattern=^1[3-9]\\d{9}$" json:"phone"`
			Age   int    `validate:"required,min=18,max=100" json:"age"`
			Email string `validate:"required,pattern=^[a-zA-Z0-9._%+\\-]+@[a-zA-Z0-9.\\-]+\\.[a-zA-Z]{2,}$" json:"email"`
		}
		v := &MultiTagTest{
			Phone: "12345678",
			Age:   15,
			Email: "invalid-email",
		}
		err := ValidateStructWithCache(v, WithLabelTag("json"))
		assert.Error(t, err)
		assert.True(t, err.Error() == "Field 'phone': format error" ||
			err.Error() == "Field 'age': value is required" ||
			err.Error() == "Field 'email': format error")
	})
}

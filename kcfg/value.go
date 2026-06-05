// value
// + 配置文件支持 $name 配置  name 为使用的变量名字 必须为 a.b.c、a 等格式
package kcfg

import (
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type Value struct {
	Bool      bool              //bool类型
	Int       int64             //整形
	String    string            // 字符串类型
	Double    float64           // 浮点数类型
	Array     []*Value          // 数组类型 []  弃用
	Object    map[string]*Value //对象类型 {} 弃用
	ValueType int               // 值类型
}

var NullValue = &Value{
	ValueType: STRING,
}

func (v *Value) GetBool() bool {
	b, err := strconv.ParseBool(v.String)
	if err != nil {
		panic(err)
	}
	return b
}

func (v *Value) GetInt() int64 {
	i, err := strconv.ParseInt(v.String, 10, 64)
	if err != nil {
		return 0
	}
	return i
}

func (v *Value) GetString() string {
	return v.String
}

func (v *Value) GetMutiLineString() string {
	AssertEqual(v.ValueType, MUTILINE_STRING, "Not String")
	return v.String
}

func (v *Value) SetString(value string) {
	//+ @2016.1.21 by kolonse
	//+ 将左右的空白处去除 如果左右 ""  那么表示该段内容就是字符串 将该段内容进行复制
	v.String = regexp.MustCompile(`^(.+?)[ \t]+$`).ReplaceAllString(value, "$1")
	v.String = regexp.MustCompile(`^[ \t]+(.+)$`).ReplaceAllString(v.String, "$1")
	v.String = regexp.MustCompile(`^"(.+)"$`).ReplaceAllString(v.String, "$1")
}

// 判断值是否包含变量类型也就是 $xxx 方式
func (v *Value) IsVars() bool {
	reg := regexp.MustCompile(`\$([a-zA-Z\._]+)`)
	return reg.MatchString(v.String)
}

// 获取到所有变量名字
func (v *Value) GetVarsName() []string {
	reg := regexp.MustCompile(`\$([a-zA-Z\._]+)`)
	vars := reg.FindAllString(v.String, len(v.String))
	var ret []string
	for _, name := range vars {
		ret = append(ret, name[1:])
	}
	return ret
}

func (v *Value) ParseVarsName(name, value string) {
	v.String = strings.ReplaceAll(v.String, "$"+name, value)
}

func NewValue(valueType int, v any) *Value {
	value := &Value{
		ValueType: valueType,
	}
	t := reflect.TypeOf(v).Kind()
	switch valueType {
	case BOOL:
		if t == reflect.Bool {
			value.Bool = v.(bool)
		} else {
			panic(errors.New("数值类型非 bool 类型,而是 " + reflect.TypeOf(v).String()))
		}
	case INT:
		if t >= reflect.Int || t <= reflect.Uint64 { // 只要是整数全部转为 int64 避免麻烦
			value.Int = v.(int64)
		} else {
			panic(errors.New("数值类型非 int 类型,而是 " + reflect.TypeOf(v).String()))
		}
	case MUTILINE_STRING:
		fallthrough
	case STRING:
		if t == reflect.String {
			value.SetString(v.(string))
			//			value.String = v.(string)
		} else {
			panic(errors.New("数值类型非 string 类型,而是 " + reflect.TypeOf(v).String()))
		}
	case DOUBLE:
		if t >= reflect.Float32 || t <= reflect.Float64 {
			value.Double = v.(float64)
		} else {
			panic(errors.New("数值类型非 float 类型,而是 " + reflect.TypeOf(v).String()))
		}
	case ARRAY:
		if t == reflect.Array {
			value.Array = append(value.Array, v.([]*Value)...)
		} else {
			panic(errors.New("数值类型非 array 类型,而是 " + reflect.TypeOf(v).String()))
		}
	case OBJECT:
		if t == reflect.Map {
			//			value.Int = v.(int64)
			value.Object = make(map[string]*Value)
			for key, vlu := range v.(map[string]*Value) {
				value.Object[key] = vlu
			}
		} else {
			panic(errors.New("数值类型非 object 类型,而是 " + reflect.TypeOf(v).String()))
		}
	}

	return value
}

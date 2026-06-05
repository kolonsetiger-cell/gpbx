/*
*
配置解析文件 配置文件格式符号说明:
# 表示注释

	关键字下方需要有一个 tab 用来表示以下的属性都属于 关键字下的列表

key:{} 表示关键字key下有一组属性是key:value的列表
key:[] 表示关键字 key的属性是一个数组
"" 表示一串字符串
*/
package kcfg

import (
	"errors"
	"log"
	"os"
	"path/filepath"
)

// 定义关键字符号
const (
	COMMENT_B = '#'
	OBJECT_S  = '{'
	OBJECT_E  = '}'
	ARRAY_S   = '['
	ARRAY_E   = ']'
	LINE_END  = "\r\n"
)

type Cfg struct {
	root    *Node
	path    string
	content string
	// vars       []*Value
	vars_nodes []*Node
}

func (cfg *Cfg) readFile(path string) string {
	cfg.path = path
	buff, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(buff)
}

func (cfg *Cfg) parseComment(content string, offset int, farther *Node) int {
	n := NewNode("#")
	i := offset + 1
	for ; i < len(content); i++ {
		// 只要碰到 \r 或者 \n 那么直接跳出
		if content[i] == '\r' || content[i] == '\n' {
			break
		}
	}
	v := NewValue(STRING, content[offset+1:i])
	n.SetValue(v)
	farther.AddChild(n)
	return i + 1
}

func (cfg *Cfg) parseObject(content string, _ string, offset int, farther *Node) int {
	stack := 1
	i := offset
	for ; stack != 0 && i < len(content); i++ {
		switch content[i] {
		case '{':
			stack++
		case '}':
			stack--
		}
	}
	if stack != 0 {
		panic(errors.New("{ 不匹配 } 数目"))
	} else {
		childString := content[offset:i]
		cfg.parseByString(childString, farther)
	}
	return i + 1
}

func (cfg *Cfg) parseArray(content string, _ string, offset int, farther *Node) int {
	stack := 1
	i := offset
	for ; stack != 0 && i < len(content); i++ {
		switch content[i] {
		case '[':
			stack++
		case ']':
			stack--
		}
	}
	if stack != 0 {
		panic(errors.New("[ 不匹配 ] 数目"))
	} else {
		childString := content[offset:i]
		cfg.parseByString(childString, farther)
	}
	return i + 1
}

func (cfg *Cfg) parseString(content string, offset int) (string, int) {
	i := offset
	for ; i < len(content); i++ {
		c := content[i]
		if '\r' == c || '\n' == c {
			return content[offset:i], i + 1
		} else if i == len(content)-1 {
			return content[offset : i+1], i + 1
		}
	}
	panic(errors.New("string 值不存在"))
}

func (cfg *Cfg) parseKey(content string, offset int) (string, int) {
	i := offset
	key := ""
	keyIndex := i
	for ; i < len(content); i++ {
		c := content[i]
		switch {
		case ARRAY_S == c:
			fallthrough
		case OBJECT_S == c:
			fallthrough
		case ' ' == c || '\t' == c:
			key = content[keyIndex:i]
			return key, i
		case '\r' == c || '\n' == c:
			// panic(errors.New("关键字没有值"))
			key = content[keyIndex:i]
			return key, i
		}
	}
	panic(errors.New("配置格式应该为[key value/{}/[]]"))
}

func (cfg *Cfg) parseValue(content string, key string, offset int) (*Node, int) {
	// 开始进行值处理
	i := offset
	for ; i < len(content); i++ {
		c := content[i]
		switch {
		case ARRAY_S == c: // 数组值
			n := NewNode(key)
			index := cfg.parseArray(content, key, i+1, n)
			return n, index
		case OBJECT_S == c: // 对象值
			n := NewNode(key)
			index := cfg.parseObject(content, key, i+1, n)
			return n, index
		case '`' == c:
			if i+2 < len(content) && content[i+1] == '`' && content[i+2] == '`' {
				// 值类型为多行类型
				index := i + 2
				end := false
				for index+2 < len(content) {
					if content[index] == '`' && content[index+1] == '`' && content[index+2] == '`' {
						end = true
						break
					}
					index++
				}

				if !end {
					panic(errors.New("值类型为多行类型，但是没有多行结束标记"))
				}

				n := NewNode(key)
				value := content[i+3 : index]
				v := NewValue(MUTILINE_STRING, value)
				if v.IsVars() {
					cfg.vars_nodes = append(cfg.vars_nodes, n)
				}
				n.SetValue(v)
				return n, index + 3
			}
			fallthrough
		case ' ' != c && '\t' != c: // 字符串值
			n := NewNode(key)
			value, index := cfg.parseString(content, i)
			v := NewValue(STRING, value)
			if v.IsVars() {
				cfg.vars_nodes = append(cfg.vars_nodes, n)
			}
			n.SetValue(v)
			return n, index
		}
	}
	panic(errors.New("配置错误 只有关键字没有值"))
}

func (cfg *Cfg) parseAttr(content string, offset int, farther *Node) int {
	key, i := cfg.parseKey(content, offset)
	n, i := cfg.parseValue(content, key, i)
	if key == "include" && n.IsString() {
		path := n.GetString()
		old := cfg.path
		path = filepath.Join(filepath.Dir(cfg.path), path)
		content := cfg.readFile(path)
		cfg.parseByString(content, farther)
		cfg.path = old
	} else {
		n.path = cfg.path
		farther.AddChild(n)
	}
	return i
}

func (cfg *Cfg) parseByString(content string, farther *Node) {
	// 遍历BUFF 对内容进行解析
	//row := 0
	for i := 0; i < len(content); {
		var offset int
		c := content[i]
		switch {
		case COMMENT_B == c:
			offset = cfg.parseComment(content, i, farther)
		case 'a' <= c && 'z' >= c:
			fallthrough
		case 'A' <= c && 'Z' >= c:
			offset = cfg.parseAttr(content, i, farther)
		default:
			offset = i + 1
		}
		i = offset
	}
}

func (cfg *Cfg) ParseFile(path string) *Cfg {
	content := cfg.readFile(path)
	return cfg.ParseByString(content)
}

func (cfg *Cfg) ParseByString(content string) *Cfg {
	cfg.content = content
	cfg.parseByString(content, cfg.root)
	// 设置环境变量
	env := cfg.Child("env")
	env_nodes_key, env_nodes_val := env.ChildsAll()
	for i, key := range env_nodes_key {
		global_env[key] = env_nodes_val[i].GetString()
	}
	// 进行一次遍历 将所有的变量进行替换成最新值

	for _, v := range cfg.vars_nodes {
		vars := v.Value.GetVarsName()
		for _, name := range vars {
			f := v.farther
			found := false

			for f != nil {
				val := f.Child(name)
				if val != nil && !val.IsNull() {
					v.ParseVarsName(name, val.GetString())
					log.Println("Var", name, "Found", val.GetString())
					found = true
					break
				} else {
					log.Println("Var", name, "Not Found, to father node")
					f = f.farther
				}
			}

			if !found {
				log.Println("Warn: Var", name, "Not Found, Use Env")
				if val, ok := global_env[name]; ok {
					v.ParseVarsName(name, val)
				} else {
					log.Println("Warn: Var", name, "Not Found From Env")
				}
			}
		}
	}

	return cfg
}

func (cfg *Cfg) Dump() string {
	//childs := cfg.root.Childs()
	return cfg.root.Dump("")
}

func (cfg *Cfg) Child(key string) *Node {
	return cfg.root.Child(key)
}

func (cfg *Cfg) Childs(key string) []*Node {
	return cfg.root.Childs(key)
}

func (cfg *Cfg) Add(key string, value string) {
	cfg.root.Add(key, value)
}

func NewCfg() *Cfg {
	return &Cfg{
		root: NewNode(""),
	}
}

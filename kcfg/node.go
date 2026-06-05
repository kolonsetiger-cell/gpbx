package kcfg

import (
	"regexp"
	"strings"
)

const (
	ONCE_CAP_LEN = 10
)

type Nodes struct {
	Ns []*Node
	l  int
	c  int
}

func (ns *Nodes) Add(n *Node) {
	if ns.l >= ns.c {
		// 需要进行扩充容器
		cn := make([]*Node, ONCE_CAP_LEN)
		ns.Ns = append(ns.Ns, cn...)
		ns.c += ONCE_CAP_LEN
	}

	ns.Ns[ns.l] = n
	ns.l++
}

func (ns *Nodes) Get() []*Node {
	return ns.Ns[:ns.l]
}

func NewNodes() *Nodes {
	return &Nodes{
		Ns: make([]*Node, ONCE_CAP_LEN),
		c:  ONCE_CAP_LEN,
		l:  0,
	}
}

type Node struct {
	key string
	*Value
	farther          *Node
	childs           map[string]*Nodes
	childs_keys      []string
	childs_keys_que  []string
	childs_nodes_que []*Node
	path             string
}

var NullNode = &Node{
	"",
	NullValue,
	nil,
	make(map[string]*Nodes),
	make([]string, 0),
	make([]string, 0),
	make([]*Node, 0),
	"",
}

func (n *Node) GetPath() string {
	return n.path
}

func (n *Node) IsNull() bool {
	return n == NullNode
}

func (n *Node) IsString() bool {
	return n.Value != nil && (n.ValueType == STRING || n.ValueType == MUTILINE_STRING)
}

func (n *Node) SetKey(key string) {
	n.key = key
}

func (n *Node) SetValue(value *Value) {
	n.Value = value
}

func (n *Node) SetFarther(farther *Node) {
	n.farther = farther
}

func (n *Node) AddChild(child *Node) *Node {
	nodes, ok := n.childs[child.key]
	if !ok { // 如果不存在该 key 那么就创建
		nodes = NewNodes()
		n.childs_keys = append(n.childs_keys, child.key)
		n.childs[child.key] = nodes
	}
	// 将该节点值添加到节点映射中
	nodes.Add(child)
	child.SetFarther(n)
	n.childs_keys_que = append(n.childs_keys_que, child.key)
	n.childs_nodes_que = append(n.childs_nodes_que, child)
	return n
}

func (n *Node) getChilds(key string) []*Node {
	nodes, ok := n.childs[key]
	if !ok {
		return []*Node{}
	}
	return nodes.Get()
}

func (n *Node) getChild(key string) *Node {
	nodes, ok := n.childs[key]
	if !ok {
		return NullNode
	}
	return nodes.Get()[0]
}

func (n *Node) Keys() []string {
	return n.childs_keys
}

func (n *Node) ChildsAll() ([]string, []*Node) {
	return n.childs_keys_que, n.childs_nodes_que
}

func (n *Node) ChildsAllWithoutNote() ([]string, []*Node) {
	keys := []string{}
	nodes := []*Node{}
	for i, k := range n.childs_keys_que {
		if k == "#" {
			continue
		}
		keys = append(keys, k)
		nodes = append(nodes, n.childs_nodes_que[i])
	}
	return keys, nodes
}

func (n *Node) Childs(key string) []*Node {
	kr := NewKeyRoute()
	kr.parse(key)
	find := func(root []*Node, k string) []*Node {
		var ret []*Node
		for _, r := range root {
			ns := r.getChilds(k)
			if ns != nil {
				ret = append(ret, ns...)
			}
		}
		return ret
	}
	nd := n.getChilds(kr.next())
	for !kr.end() && nd != nil {
		nd = find(nd, kr.next())
	}
	return nd
}

func (n *Node) Child(key string) *Node {
	kr := NewKeyRoute()
	kr.parse(key)
	find := func(root *Node, k string) *Node {
		return root.getChild(k)
	}
	nd := n.getChild(kr.next())
	for !kr.end() && nd != nil {
		nd = find(nd, kr.next())
	}
	return nd
}

func (n *Node) Add(key string, value string) {
	kr := NewKeyRoute()
	kr.parse(key)
	cur_node := n
	for {
		k := kr.next()
		nd := cur_node.getChild(k)
		if nd == NullNode {
			// 如果是 Null Node 需要创建节点
			nd = NewNode(k)
			cur_node.AddChild(nd)
			if kr.end() {
				// 如果已经是最后一个节点，设置值结束
				var v *Value
				if strings.Contains(value, "\n") {
					v = NewValue(MUTILINE_STRING, value)
				} else {
					v = NewValue(STRING, value)
				}
				nd.SetValue(v)
				break
			} else {
				cur_node = nd
				continue
			}
		} else {
			if kr.end() {
				v := NewValue(STRING, value)
				nd.SetValue(v)
				break
			} else {
				cur_node = nd
				continue
			}
		}
	}
}

func (n *Node) Dump(suffix string) string {
	ret := ""
	// node 值不为空时进行打印
	if n.key != "" && n.Value != nil && n.Value.ValueType != INVALID {
		if n.Value.ValueType == MUTILINE_STRING {
			str := n.Value.GetMutiLineString()
			ret += suffix + n.key + " ```" + str + "\n```\n"
		} else {
			str := n.Value.GetString()
			// 如果字符串两边只要有一端有空格 那么就需要增加 ""
			if regexp.MustCompile(`^((([ \t]+)?.+[ \t]+)|([ \t]+.+([ \t]+)?))$`).MatchString(str) {
				str = "\"" + str + "\""
			}
			ret += suffix + n.key + " " + str + "\n"
		}

	} else if n.key != "" {
		ret += suffix + n.key + " {\n"
		for i := 0; i < len(n.childs_keys_que); i++ {
			node := n.childs_nodes_que[i]
			ret += node.Dump(suffix + "\t")
		}
		ret += suffix + "}\n"
	} else {
		for i := 0; i < len(n.childs_keys_que); i++ {
			node := n.childs_nodes_que[i]
			ret += node.Dump(suffix + "\t")
		}
	}
	return ret
}

func (n *Node) AddChilds(childs []*Node) {
	//	n.child = append(n.child, childs...)
	//	for _, i := range childs {
	//		childs[i].SetFarther(n)
	//	}
}

func NewNode(key string) *Node {
	return &Node{
		key:    key,
		childs: make(map[string]*Nodes),
	}
}

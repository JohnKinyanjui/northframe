package compiler

type component struct {
	Name     string
	Imports  []componentImport
	Props    []prop
	Style    string
	Children []node
}

type componentImport struct {
	Alias string
	Path  string
}

type prop struct {
	Name string
	Type string
}

type node interface {
	node()
}

type textNode struct{ Value string }
type exprNode struct{ Expression string }
type styleNode struct{ Value string }
type slotNode struct{}
type propsNode struct{}
type ifNode struct {
	Condition string
	Children  []node
}
type componentNode struct {
	Name       string
	Attributes []componentAttribute
	Children   []node
}

type componentAttribute struct {
	Name    string
	Value   string
	Literal bool
	Quoted  bool
}
type eachNode struct {
	Collection string
	Item       string
	Children   []node
}

func (textNode) node()      {}
func (exprNode) node()      {}
func (styleNode) node()     {}
func (slotNode) node()      {}
func (propsNode) node()     {}
func (ifNode) node()        {}
func (eachNode) node()      {}
func (componentNode) node() {}

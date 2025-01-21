package analysis

import "borm-lsp/lsp"

type SynType string

const (
	TEXT SynType = "Text"
	METHOD = "method"
	FUNCTION = "function"
	CONSTRUCTOR = "constructor" 
	FIELD = "field" 
	VARIABLE = "variable" 
	CLASS = "class" 
	INTERFACE = "interface" 
	MODULE = "module" 
	PROPERTY = "property" 
	UNIT = "unit" 
	VALUE = "value" 
	ENUM = "enum" 
	KEYWORD = "keyword" 
	SNIPPET = "snippet" 
	COLOR = "color" 
	FILE = "file" 
	REFERENCE = "reference" 
	FOLDER = "folder" 
	ENUMMEMBER = "enummember" 
	CONSTANT = "constant" 
	STRUCT = "struct" 
	EVENT = "event" 
	OPERATOR = "operator" 
	TYPEPARAMETER = "typeparameter" 
	BLOCK = "block" 
	COMMENT = "comment" 
)

type SyntaxNode struct {
	Children []SyntaxNode
	Parent *SyntaxNode
	Start lsp.Position
	End lsp.Position
	Value string
	Type SynType
}

func NewNode(par *SyntaxNode, val string, t SynType, start, end lsp.Position) SyntaxNode {
	return SyntaxNode{
		Children: []SyntaxNode{},
		Parent: par,
		Value: val,
		Type: t,
		Start: start,
		End: end,
	}
}

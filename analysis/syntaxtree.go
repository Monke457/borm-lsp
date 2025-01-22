package analysis

import (
	"borm-lsp/lsp"
	"log"
	"strings"
)

type SynType string

const (
	TEXT SynType = "text"
	FUNCTION = "function"
	FIELD = "field" 
	VARIABLE = "variable" 
	CLASS = "class" 
	INTERFACE = "interface" 
	VALUE = "value" 
	ENUM = "enum" 
	KEYWORD = "keyword" 
	FILE = "file" 
	ENUMMEMBER = "enummember" 
	CONSTANT = "constant" 
	TYPEPARAMETER = "typeparameter" 
	BLOCK = "block" 
	COMMENT = "comment" 
	INCLUDE = "include" 
)

type SyntaxNode struct {
	Children []SyntaxNode
	Parent *SyntaxNode
	Start lsp.Position
	End lsp.Position
	Value string
	Type SynType
	IsBad bool
}

func NewNode(par *SyntaxNode, val string, t SynType, start, end lsp.Position) SyntaxNode {
	return SyntaxNode{
		Children: []SyntaxNode{},
		Parent: par,
		Start: start,
		End: end,
		Value: val,
		Type: t,
		IsBad: false,
	}
}

func (n SyntaxNode) FindNodeAtPosition(logger *log.Logger, pos lsp.Position) (SyntaxNode, bool) {
	if n.Start.Line == pos.Line && n.End.Line == pos.Line {
		if n.Start.Character <= pos.Character && n.End.Character >= pos.Character {
			return n, true
		}
		return n, false
	}
	closest := n
	for _, child := range n.Children {
		if child.Start.Line > pos.Line || child.End.Line < pos.Line {
			continue
		}
		closest = child
		if _, found := child.FindNodeAtPosition(logger, pos); found {
			return child, true
		}
	}
	return closest, false
}

func (n SyntaxNode) GetBadNodes() []SyntaxNode {
	results := []SyntaxNode{}
	if n.IsBad {
		results = append(results, n)
	}
	if len(n.Children) == 0 {
		return results
	}
	for _, child := range n.Children {
		results = append(results, child.GetBadNodes()...)
	}
	return results
}

func CreateTree(logger *log.Logger, doc, text string) SyntaxNode {
	tokens := Tokenize(text)
	if len(tokens) == 0 {
		return SyntaxNode{}
	}

	root := NewNode(nil, doc, FILE, GetStartPos(tokens...), GetFinalPos(tokens...))

	return root.createBranch(logger, tokens)
}

func (n SyntaxNode) createBranch(logger *log.Logger, tokens []Token) SyntaxNode {
	i := 0
	for i < len(tokens) {
		token := tokens[i]
		if len(token.value) < 2 {
			i++ 
			continue
		}
		if token.value[:2] == "//" {
			//it's a comment
			node, jump := createCommentNode(tokens[i:])
			node.Parent = &n
			n.Children = append(n.Children, node)
			i += jump+1
			continue
		}
		if token.value == "#include" {
			//it's an include statement 
			node, jump := createIncludeNode(tokens[i:])
			node.Parent = &n
			n.Children = append(n.Children, node)
			i += jump+1
			continue
		}
		if token.value == "function" {
			/*
			// it's function
			node := createBlockNode(tokens[i:])
			node.Parent = &n
			n.Children = append(n.Children, node)
			i = node.End.Line
			line = lines[i]
			charPos = node.End.Character
			*/
		}
		// unhandled/text token
		node := NewNode(&n, token.value, TEXT, token.pos, GetFinalPos(token))
		n.Children = append(n.Children, node)
		i++
	}
	return n
}

func createCommentNode(tokens []Token) (SyntaxNode, int) {
	tokens = GetTokensToNewLine(tokens)
	spent := len(tokens)-1
	value := Stringify(tokens)
	finalPos := GetFinalPos(tokens...)
	node := NewNode(nil, value, COMMENT, tokens[0].pos, finalPos)
	return node, spent
}

func createIncludeNode(tokens []Token) (SyntaxNode, int) {
	tokens = GetTokensToNewLine(tokens)
	spent := len(tokens)-1
	finalPos := GetFinalPos(tokens...)
	value := Stringify(tokens)
	node := NewNode(nil, value, INCLUDE, tokens[0].pos, finalPos)

	keywordNode := NewNode(&node, "include", KEYWORD, tokens[0].pos, GetFinalPos(tokens[0]))
	node.Children = append(node.Children, keywordNode)
	
	if len(tokens) < 2 {
		node.IsBad = true
		return node, spent
	}
	if len(tokens) > 2 {
		node.IsBad = true
	}

	valueNode := NewNode(&node, "", VALUE, tokens[1].pos, GetFinalPos(tokens[1]))
	switch tokens[1].value[0] {
	case '"':
		if tokens[1].value[len(tokens[1].value)-1] != '"' {
			valueNode.IsBad = true
		}
	case '<':
		if tokens[1].value[len(tokens[1].value)-1] != '>' {
			valueNode.IsBad = true
		}
	default:
		valueNode.IsBad = true
	}
	valueNode.Value = strings.Trim(tokens[1].value, "\"<>")

	node.Children = append(node.Children, valueNode)
	return node, spent
}

func createFileNode(text string, line, charPos int) SyntaxNode {
	value, _, _ := strings.Cut(text[charPos:], " ") 
	value = strings.TrimSpace(value)
	encloser := value[0]

	value = strings.Trim(value, "\"<>'")

	start := strings.Index(text, value)
	node:= NewNode(
		nil, 
		value, 
		FILE, 
		lsp.Position{Line:line, Character:start}, 
		lsp.Position{Line:line, Character:start+len(value)}, 
	) 
	if len(value) == 0 || 
	!(encloser == '"' || encloser == '<' || encloser == '\'') ||
	(encloser == '<' && !strings.Contains(text[charPos:], ">")) {
		node.IsBad = true
	}
	return node
}

package analysis

import (
	"borm-lsp/lsp"
	"fmt"
	"log"
	"strings"
)

type SynType string

const (
	TEXT SynType = "text"
	FUNCTION = "function"
	DECLARATION = "declaration"
	NAME = "name"
	FIELD = "field" 
	VARIABLE = "variable" 
	VALUE = "value" 
	KEYWORD = "keyword" 
	FILE = "file" 
	BLOCK = "block" 
	COMMENT = "comment" 
	INCLUDE = "include" 
	TYPE = "type"
)

type SyntaxNode struct {
	Children []SyntaxNode
	Parent *SyntaxNode
	Start lsp.Position
	End lsp.Position
	Value string
	Type SynType
	IsBad bool
	Diagnostics []string
}

func NewNode(parent *SyntaxNode, start, end lsp.Position, value string, t SynType) SyntaxNode {
	return SyntaxNode {
		Children: []SyntaxNode{}, 
		Parent: parent,
		Start: start, 
		End: end,
		Value: value, 
		Type: t, 
		IsBad: false,
		Diagnostics: []string{},
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

	root := SyntaxNode{
		Value:doc, 
		Type: FILE, 
		Start: GetStartPos(tokens...), 
		End: GetFinalPos(tokens...),
	}

	return root.createBranch(logger, tokens)
}

func (n SyntaxNode) passedHeader() bool {
	for _, child := range n.Children {
		if child.Type != COMMENT && child.Type != INCLUDE {
			return true
		}
	}
	return false
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
		switch token.value {
		case "#include":
			//it's an include statement 
			node, jump := createIncludeNode(tokens[i:])
			node.Parent = &n
			if n.Type != FILE || n.passedHeader() {
				// is bad
				node.IsBad = true
				node.Diagnostics = append(node.Diagnostics, "include statements must appear before other declarations.")
			} 
			n.Children = append(n.Children, node)
			i += jump
		case "void":
			// it's a function
			node, jump := createFunctionNode(tokens[i:])
			node.Parent = &n
			n.Children = append(n.Children, node)
			i += jump
		case "int", "double", "float", "char", "bool", "string":
			//it's a declaration. could be a function
			node, jump := createDeclarationNode(tokens[i:])
			node.Parent = &n
			n.Children = append(n.Children, node)
			i += jump

		default:
			// unhandled/text token
			node := NewNode(&n, token.pos, GetFinalPos(token), token.value, TEXT)
			n.Children = append(n.Children, node)
			i++
		}
	}
	return n
}

func createDeclarationNode(tokens []Token) (SyntaxNode, int) {
	if len(tokens) == 1 {
		//is bad nothing after type
		node := NewNode(nil, GetStartPos(tokens[0]), GetFinalPos(tokens[0]), tokens[0].value, TYPE)
		node.IsBad = true
		node.Diagnostics = append(node.Diagnostics, fmt.Sprintf("%s (type) is not an expression", tokens[0].value))
		return node, 1
	}

	if tokens[1].value == "function" {
		// it's a function
		return createFunctionNode(tokens)
	}

	//it's a variable
	return createVariableNode(tokens)
}

func createFunctionNode(tokens []Token) (SyntaxNode, int) {
	firstLine := GetTokensToNewLine(tokens)
	headNode, name, err := createFunctionHeadNode(firstLine)
	if err != nil {
		// is bad, function not properly formed
		headNode.IsBad = true
		headNode.Diagnostics = append(headNode.Diagnostics, err.Error())
	}

	node := NewNode(nil, GetStartPos(firstLine...), headNode.End, name, FUNCTION)
	node.Children = append(node.Children, headNode)
	if headNode.Type == DECLARATION {
		return node, len(firstLine)
	}

	bodyNode, err := createFunctionBodyNode(tokens[len(firstLine):])
	if err != nil {
		// is bad, function not properly formed
		bodyNode.IsBad = true
		bodyNode.Diagnostics = append(bodyNode.Diagnostics, "Malformed function")
	}
	node.Children = append(node.Children, bodyNode)

	return node, len(firstLine)
}

func createFunctionHeadNode(tokens []Token) (SyntaxNode, string, error) {
	node := NewNode(nil, GetStartPos(tokens...), GetFinalPos(tokens...), Stringify(tokens), DECLARATION)
	if len(tokens) == 1 {
		return node, "", fmt.Errorf("%s is not an expression", tokens[0].value)		
	}
	typeNode := NewNode(&node, GetStartPos(tokens[0]), GetFinalPos(tokens[0]), tokens[0].value, TYPE)
	node.Children = append(node.Children, typeNode)

	if tokens[1].value != "function" {
		return node, "", fmt.Errorf("Malformed function declaration")		
	}
	keyWordNode := NewNode(&node, GetStartPos(tokens[1]), GetFinalPos(tokens[1]), tokens[1].value, KEYWORD)
	node.Children = append(node.Children, keyWordNode)

	if len(tokens) == 2 {
		return node, "", fmt.Errorf("A function must have a name")
	}

	if !strings.Contains(tokens[2].value, "(") {
		//deal with this later i guess
	}

	name, params, _ := strings.Cut(tokens[2].value, "(")
	start := GetStartPos(tokens[2])
	end := lsp.Position{Line:start.Line, Character:start.Character + len(name)}
	nameNode := NewNode(&node, start, end, name, NAME)

	params, rest, found := strings.Cut(params, ")")
	if found {
		// we have no params
		if strings.TrimSpace(rest)[0] == ';' {
			// it's a declaration
			return node, name, nil
		} 
		node.Type = FUNCTION 
		return node, name, nil
	}

	paramNodes := []SyntaxNode{}
	if len(params) > 0 {
		paramstart := lsp.Position{Line:start.Line, Character: end.Character+2}
		paramend := lsp.Position{Line:start.Line, Character: paramstart.Character+len(params)}
		paramNodes = append(paramNodes, NewNode(&node, paramstart, paramend, params, TYPE))
	}

	node.Children = append(node.Children, nameNode)
	node.Children = append(node.Children, paramNodes...)

	if len(tokens) == 3 {
		return node, "", fmt.Errorf("Malformed function.")
	}

	return node, nameNode.Value, nil
}

func createVariableNode(tokens []Token) (SyntaxNode, int) {
	node := NewNode(nil, GetStartPos(tokens...), GetFinalPos(tokens...), Stringify(tokens), VARIABLE)
	return node, 0
}

func createCommentNode(tokens []Token) (SyntaxNode, int) {
	tokens = GetTokensToNewLine(tokens)
	node := NewNode(nil, GetStartPos(tokens...), GetFinalPos(tokens...), Stringify(tokens), COMMENT)
	return node, len(tokens)
}

func createIncludeNode(tokens []Token) (SyntaxNode, int) {
	tokens = GetTokensToNewLine(tokens)
	spent := len(tokens)
	node := NewNode(nil, tokens[0].pos, GetFinalPos(tokens...), Stringify(tokens), INCLUDE)

	keywordNode := NewNode(&node, tokens[0].pos, GetFinalPos(tokens[0]), "include", KEYWORD)
	node.Children = append(node.Children, keywordNode)
	
	if len(tokens) < 2 {
		node.IsBad = true
		node.Diagnostics = append(node.Diagnostics, "You must supply a file or header to be included.")
		return node, spent
	}
	if len(tokens) > 2 {
		node.IsBad = true
		node.Diagnostics = append(node.Diagnostics, "Syntax error. Please attach a single header or file per include.")
	}

	valueNode := NewNode(&node, tokens[1].pos, GetFinalPos(tokens[1]), "", VALUE)
	switch tokens[1].value[0] {
	case '"':
		if tokens[1].value[len(tokens[1].value)-1] != '"' {
			valueNode.IsBad = true
			valueNode.Diagnostics = append(valueNode.Diagnostics, "Unclosed quotation marks.")
		}
	case '<':
		if tokens[1].value[len(tokens[1].value)-1] != '>' {
			valueNode.IsBad = true
			valueNode.Diagnostics = append(valueNode.Diagnostics, "Unclosed angle brackets.")
		}
	default:
		valueNode.IsBad = true
		valueNode.Diagnostics = append(valueNode.Diagnostics, "Syntax error. Accepted include formats: <header.h> or \"file_path\".")
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


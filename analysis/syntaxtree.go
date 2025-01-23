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
	NAMESPACE = "namespace"
	FIELD = "field" 
	VARIABLE = "variable" 
	VALUE = "value" 
	KEYWORD = "keyword" 
	FILE = "file" 
	REFERENCE = "reference"
	HEADERFILE = "header_file" 
	BLOCK = "block" 
	COMMENT = "comment" 
	INCLUDE = "include" 
	TYPE = "type"
)

//TODO: this needs to be better
//values in the nodes are wrong
//since nodes can be made of multiple tokens and the whitespace is lost
//File paths COULD potentially contain spaces
//Comments LIKELY to contain whitespace

//could fix this at the tokenizer level
//I.e., take into account quotes and comments when tokenizing instead of at tree level logic
//downside is that we will lose granularity 
//(individual nodes within comments or quotes are lost, single token for all)
//meaning hypertext comments or string placeholders are not analysed

//another solution.. not storing all values in nodes: do we need to store them here?
//we can find the actual values in the files using position info
//this is probably the better solution

//for now work without values and see if any issues come up
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
		return NewNode(nil, lsp.Position{}, lsp.Position{}, doc, FILE)
	}

	root := NewNode(nil, GetStartPos(tokens...), GetFinalPos(tokens...), doc, FILE)

	return root.createBranch(logger, tokens)
}

func (n SyntaxNode) passedHeader() bool {
	if n.Type != FILE {
		return true
	}
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
		logger.Printf("looping tokens for node type: %s. current children %d", n.Type, len(n.Children))
		token := tokens[i]
		switch token.value {
		case "/": 
			if i >= len(tokens) || tokens[i+1].value != "/" {
				// it's just a weird slash???
				// this is probably bad
				i++
				break
			}
			//it's a comment
			node, jump := createCommentNode(tokens[i:])
			node.Parent = &n
			n.Children = append(n.Children, node)
			logger.Printf("added %s", node.Type)
			i += jump
		case ";", ":", "\"", "'", "`", "(", ")", "{", "}" :
			// it's something unexpected (should be handled in other cases
			// probably an issue...)
			i++

		case "#include":
			//it's an include statement 
			node, jump := createIncludeNode(tokens[i:])
			node.Parent = &n
			if n.passedHeader() {
				// is bad
				node.IsBad = true
				node.Diagnostics = append(node.Diagnostics, "include statements must appear before other declarations.")
			} 
			i += jump
			n.Children = append(n.Children, node)
			logger.Printf("added %s", node.Type)
		case "void":
			// it's a function
			node, jump := createFunctionNode(tokens[i:])
			node.Parent = &n
			n.Children = append(n.Children, node)
			i += jump
			logger.Printf("added %s", node.Type)
		case "int", "double", "float", "char", "bool", "string":
			//it's a declaration. could be a function
			node, jump := createDeclarationNode(tokens[i:])
			node.Parent = &n
			n.Children = append(n.Children, node)
			i += jump
			logger.Printf("added %s", node.Type)

		default:
			// unhandled/text token
			node := NewNode(&n, token.pos, GetFinalPos(token), token.value, TEXT)
			n.Children = append(n.Children, node)
			i++
			logger.Printf("added %s: value %s", node.Type, node.Value)
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
	headNode, err := createFunctionHeadNode(firstLine)
	if err != nil {
		// is bad, function not properly formed
		headNode.IsBad = true
		headNode.Diagnostics = append(headNode.Diagnostics, err.Error())
	}

	node := NewNode(nil, GetStartPos(firstLine...), headNode.End, "", FUNCTION)
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

func createFunctionHeadNode(tokens []Token) (SyntaxNode, error) {
	node := NewNode(nil, tokens[0].pos, tokens[1].pos, "", FUNCTION)
	return node, nil
}

func createFunctionBodyNode(tokens []Token) (SyntaxNode, error) {
	node := NewNode(nil, tokens[0].pos, tokens[1].pos, "", FUNCTION)
	return node, nil
}

func createVariableNode(tokens []Token) (SyntaxNode, int) {
	node := NewNode(nil, GetStartPos(tokens...), GetFinalPos(tokens...), Stringify(tokens), VARIABLE)
	return node, 0
}

func createCommentNode(tokens []Token) (SyntaxNode, int) {
	tokens = GetTokensToNewLine(tokens)
	node := NewNode(nil, GetStartPos(tokens...), GetFinalPos(tokens...), "", COMMENT)
	return node, len(tokens)
}

func createIncludeNode(tokens []Token) (SyntaxNode, int) {
	// #include <header.h>
	// #include "G://dir/file_path.sct"
	var node SyntaxNode
	children := []SyntaxNode{}

	var endVal string
	line := tokens[0].pos.Line
	count := 0
	for _, token := range IterTokensToAnyValue(tokens, "<", "\"") {
		if token.pos.Line != line {
			break
		}
		switch token.value {
		case "#include":
			n := NewNode(&node, GetStartPos(token), GetFinalPos(token), "", KEYWORD)
			if count != 0 {
				n.IsBad = true
				n.Diagnostics = append(n.Diagnostics, "Unexpected '#include' token position")
			}
			children = append(children, n)
		case "<":
			endVal = ">"
		case "\"": 
			endVal = "\""
		default:
			n := NewNode(&node, GetStartPos(token), GetFinalPos(token), "", REFERENCE)
			n.IsBad = true
			n.Diagnostics = append(n.Diagnostics, "Unexpected token")
			children = append(children, n)
		}
		count++
	}
	//this ends ON the index of the opening token 
	//tokens[count].value == "\"" or "<"
	//do not count this as start position of reference (the next one is the start)

	valueCount := 0
	for _, token := range IterTokensToValue(tokens[count:], endVal) {
		if token.pos.Line != line {
			break
		}
		valueCount++
	}
	//this ends ON the index of the end token
	//do not count this as final position of reference (the previous is sthe start)

	if valueCount > 0 {
		//TODO: make this better... it's a little hard to understand all the indexes
		refnode := NewNode(&node, GetStartPos(tokens[count]), GetFinalPos(tokens[count+valueCount-1]), "", REFERENCE)
		if endVal == ">" && valueCount != 4 {
			refnode.IsBad = true
			refnode.Diagnostics = append(refnode.Diagnostics, "Malformed header file")
		}
		children = append(children, refnode)
	} 
	count += valueCount

	if count < len(tokens) && tokens[count+1].pos.Line == line+1 && tokens[count+1].value == "using" {
		// has a namespace
		nsNode, nsCount := createNamespaceNode(tokens[count+1:])
		nsNode.Parent = &node
		children = append(children, nsNode)
		count += nsCount
	}

	node = NewNode(nil, tokens[0].pos, GetFinalPos(tokens[count]), "", INCLUDE)
	node.Children = children
	return node, count
}

func createNamespaceNode(tokens []Token) (SyntaxNode, int) {
	var node SyntaxNode
	children := []SyntaxNode{}

	line := tokens[0].pos.Line
	count := 0
	for _, token := range IterTokensToValue(tokens, ";") {
		if token.pos.Line != line {
			break
		}

		switch token.value {
		case "using":
			n := NewNode(&node, GetStartPos(token), GetFinalPos(token), "", KEYWORD)
			if count != 0 {
				n.IsBad = true
				n.Diagnostics = append(n.Diagnostics, "Unexpected 'using' token position")
			}
			children = append(children, n)

		case "namespace":
			n := NewNode(&node, GetStartPos(token), GetFinalPos(token), "", KEYWORD)
			if count != 1 {
				n.IsBad = true
				n.Diagnostics = append(n.Diagnostics, "Unexpected 'namespace' token position")
			}
			children = append(children, n)

		default:
			n := NewNode(&node, GetStartPos(token), GetFinalPos(token), "", REFERENCE)
			if count != 2 {
				n.IsBad = true
				n.Diagnostics = append(n.Diagnostics, "Unexpected token")
			}
			children = append(children, n)
		}
		count++
	}

	node = NewNode(nil, tokens[0].pos, GetFinalPos(tokens...), "", NAMESPACE)
	node.Children = children

	return node, count
}

func createFileNode(text string, line, charPos int) SyntaxNode {
	value, _, _ := strings.Cut(text[charPos:], " ") 
	value = strings.TrimSpace(value)
	encloser := value[0]

	value = strings.Trim(value, "\"<>'")

	start := strings.Index(text, value)
	node:= NewNode(
		nil, 
		lsp.Position{Line:line, Character:start}, 
		lsp.Position{Line:line, Character:start+len(value)}, 
		value, 
		FILE, 
	) 
	if len(value) == 0 || 
	!(encloser == '"' || encloser == '<' || encloser == '\'') ||
	(encloser == '<' && !strings.Contains(text[charPos:], ">")) {
		node.IsBad = true
	}
	return node
}


package analysis

import (
	"borm-lsp/lsp"
	"log"
	"strings"
)

type SyntaxTree struct {
	Root *SyntaxNode
	Document string
}

func (s SyntaxTree) FindNodeAtPosition(logger *log.Logger, pos lsp.Position) (SyntaxNode, bool) {
	return s.Root.recursiveSearch(logger, pos)
}

func (n SyntaxNode) recursiveSearch(logger *log.Logger, pos lsp.Position) (SyntaxNode, bool) {
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
		if _, found := child.recursiveSearch(logger, pos); found {
			return child, true
		}
	}
	return closest, false
}

func CreateTree(logger *log.Logger, doc, text string) SyntaxTree {
	lines := strings.Split(text, "\n")
	start := lsp.Position{Line:0, Character:0}
	end := lsp.Position{Line:len(lines), Character:len(lines[len(lines)-1])}
	logger.Println("creating a tree of doc", doc, text, start, end)
	root := NewNode(nil, doc, FILE, start, end)
	tree := SyntaxTree{Root: &root}

	for i, line := range lines {
		if idx := strings.Index(line, "//"); idx >= 0 {
			//it's a comment
			start := lsp.Position{Line:i, Character:idx}
			end := lsp.Position{Line:i, Character:len(line)}
			comment := NewNode(&root, line[idx:], COMMENT, start, end)
			root.Children = append(root.Children, comment)
			logger.Println("found and added a child comment node", comment)
			continue
		}
		if idx := strings.Index(line, "{"); idx >= 0 {
			//it's a block of some sort
			endPos, found := getClosingBracket(lines, i)
			if !found {
				continue
			}
			start := lsp.Position{Line:i, Character:idx}
			block := NewNode(&root, "", BLOCK, start, endPos)
			root.Children = append(root.Children, block)
			logger.Println("found and added a child block node", block)
		}
	}

	logger.Println("created tree with root node", root)
	return tree
}

func getClosingBracket(text []string, line int) (lsp.Position, bool) {
	pos := lsp.Position{}
	depth := 1
	for i, next := range text[line:] {
		if comment := strings.Index(next, "//"); comment >= 0 {
			if comment == 0 {
				continue
			}
			next = next[:comment] 
		}

		c := strings.Count(next, "{")
		if c > 0 && i > 0 {
			c -= 1
		}
		depth += c

		for idx := strings.Index(next, "}"); idx >= 0; {
			depth--
			if depth == 0 {
				pos.Line = line + i
				pos.Character = strings.Index(next, "}")
				return pos, true
			}
			next = next[idx:]
		}
	}
	return  pos, false
}

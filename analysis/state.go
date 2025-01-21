package analysis

import (
	"borm-lsp/lsp"
	"fmt"
	"log"
)

type State struct {
	Documents map[string]SyntaxTree
}

func NewState() State {
	return State{
		Documents:map[string]SyntaxTree{}, 
	}
}

func getDiagnosticsForFile(text string) []lsp.Diagnostic {
	diagnostics := []lsp.Diagnostic{}

	diagnostics = append(diagnostics, lsp.Diagnostic{
		Range: LineRange(0, 0, 0),
		Severity: 1,
		Source: "Base Library",
		Message: "diagnostics operational",
	})
	
	return diagnostics
}

func (s *State) OpenDocument(logger *log.Logger, uri, text string) []lsp.Diagnostic {
	s.Documents[uri] = CreateTree(logger, uri, text)
	return getDiagnosticsForFile(text)
}

func (s *State) UpdateDocument(logger *log.Logger, uri, text string) []lsp.Diagnostic {
	s.Documents[uri] = CreateTree(logger, uri, text)
	return getDiagnosticsForFile(text)
}

func (s *State) Hover(logger *log.Logger, id int, uri string, position lsp.Position) lsp.HoverResponse {
	node, found := s.Documents[uri].FindNodeAtPosition(logger, position)
	var content string
	if found {
		content = fmt.Sprintf("Found a node! Type: %s Value: %s", node.Type, node.Value)
	} else {
		content = fmt.Sprintf("Closest node type: %s Value: %s", node.Type, node.Value)
	}
	return lsp.HoverResponse {
		Response: lsp.Response {
			RPC: "2.0",
			Id: &id,
		},
		Result: lsp.HoverResult {
			Contents: content,
		},
	}
}

func (s *State) Definition(id int, uri string, position lsp.Position) lsp.DefinitionResponse {
	//TODO: correct implementation for go-to-definition
	return lsp.DefinitionResponse {
		Response: lsp.Response {
			RPC: "2.0",
			Id: &id,
		},
		Result: lsp.Location {
			URI: uri,
			Range: lsp.Range{
				Start: lsp.Position {
					Line: position.Line - 1,
					Character: 0,
				},
				End: lsp.Position {
					Line: position.Line - 1,
					Character: 0,
				},
			},
		},
	}
}

func (s *State) CodeAction(id int, uri string) lsp.CodeActionResponse {
	actions := []lsp.CodeAction{}

	return lsp.CodeActionResponse {
		Response: lsp.Response {
			RPC: "2.0",
			Id: &id,
		},
		Result: actions,
	}
}

func (s *State) Completion(id int, uri string) lsp.CompletionResponse {
	items := []lsp.CompletionItem{{
		Label: "Deez nuts",
		Kind: 1,
		Detail: "Deez nuts in your mouth",
		Documentation: "Skibideez",
		AdditionalTextEdits: []lsp.TextEdit{{
			Range: LineRange(0, 0, 0),
			NewText: "#include \"deez\"\n",
		}},
	}}
	
	response := lsp.CompletionResponse {
		Response: lsp.Response {
			RPC: "2.0",
			Id: &id,
		},
		Result: items,
	}
	return response
}

func LineRange(line, start, end int) lsp.Range {
	return lsp.Range{
		Start: lsp.Position{
			Line: line,
			Character: start,
		},
		End: lsp.Position{
			Line: line,
			Character: end,
		},
	}
}

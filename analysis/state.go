package analysis

import (
	"borm-lsp/lsp"
	"fmt"
	"strings"
)

type State struct {
	Documents map[string]string
	CompletionItems []lsp.CompletionItem
}

func NewState(datafile string) (State, error) {
	completionItems, err := initCompletionItems(datafile)
	return State{
		Documents:map[string]string{},
		CompletionItems: completionItems,
	}, err
}

func initCompletionItems(datafile string) ([]lsp.CompletionItem, error) {
	functions, err := ReadFunctionsFromFile(datafile)
	items := []lsp.CompletionItem{}
	if err != nil {
		return items, err
	}
	for _, function := range functions {
		items = append(items, lsp.CompletionItem{
			Label: function.Name, 
			Kind: 3,
			Detail: function.Definition,
			Documentation: function.Description,
		})
	}
	return items, nil
}

func getDiagnosticsForFile(text string) []lsp.Diagnostic {
	diagnostics := []lsp.Diagnostic{}

	for row, line := range strings.Split(text, "\n") {
		if idx := strings.Index(line, "function"); idx >= 0 {
			paren := strings.Index(line, "(")
			if paren < 0 {
				diagnostics = append(diagnostics, lsp.Diagnostic{
					Range: LineRange(row, idx, len(line)),
					Severity: 1,
					Source: "Base Library",
					Message: "A function must have a defined body within parentheses: void function main(args...) {}",
				})	
				continue
			}
			if len(strings.TrimSpace(line[idx+len("function"):paren])) == 0 {
				diagnostics = append(diagnostics, lsp.Diagnostic{
					Range: LineRange(row, idx, paren),
					Severity: 1,
					Source: "Base Library",
					Message: "Functions must be named: void function main(args...) {}",
				})	
			}
		}
	}
	
	return diagnostics
}

func (s *State) OpenDocument(uri, text string) []lsp.Diagnostic {
	s.Documents[uri] = text
	return getDiagnosticsForFile(text)
}

func (s *State) UpdateDocument(uri, text string) []lsp.Diagnostic {
	s.Documents[uri] = text
	return getDiagnosticsForFile(text)
}

func (s *State) Hover(id int, uri string, position lsp.Position) lsp.HoverResponse {
	document := s.Documents[uri]

	return lsp.HoverResponse {
		Response: lsp.Response {
			RPC: "2.0",
			Id: &id,
		},
		Result: lsp.HoverResult {
			Contents: fmt.Sprintf("File: %s, Characters: %d", uri, len(document)),
		},
	}
}

func (s *State) Definition(id int, uri string, position lsp.Position) lsp.DefinitionResponse {
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
	text := s.Documents[uri]

	actions := []lsp.CodeAction{}
	openRow, openIdx, closeIdx := -1, -1, -1
	var openText string
	for row, line := range strings.Split(text, "\n") {
		if idx := strings.Index(line, "OpenTable("); idx >= 0 {
			openRow = row
			openIdx = idx
			openText = line
		}
		if idx := strings.Index(line, "CloseTable("); idx >= 0 {
			closeIdx = idx
		}
		if idx1, idx2 := strings.Index(line, "}"), strings.Index(line, "return"); idx1 >= 0 || idx2 >= 0 {
			if openIdx < 0 || closeIdx > -1 {
				closeIdx = -1
				openIdx = -1
				continue
			} 
			closeText := createCloseText(openText)
			insertChange := map[string][]lsp.TextEdit{}
			insertChange[uri] = []lsp.TextEdit{
				{
					Range: LineRange(row, 0, 0),
					NewText: closeText,
				},
			}

			actions = append(actions, lsp.CodeAction{
				Title: fmt.Sprintf("Close unclosed database connection on row %d: %s", openRow, openText),
				Edit: &lsp.WorkspaceEdit{Changes: insertChange},
			})

			openIdx, closeIdx, closeText = -1, -1, ""
		}
	}

	return lsp.CodeActionResponse {
		Response: lsp.Response {
			RPC: "2.0",
			Id: &id,
		},
		Result: actions,
	}
}

func (s *State) Completion(id int, uri string) lsp.CompletionResponse {
	response := lsp.CompletionResponse {
		Response: lsp.Response {
			RPC: "2.0",
			Id: &id,
		},
		Result: s.CompletionItems,
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

func createCloseText(line string) string {
	var text string
	if idx := strings.Index(line, ","); idx >= 0 {
		text = line[:idx]
	} else if idx := strings.Index(line, ")"); idx >= 0 {
		text = line[:idx]
	}
	text = strings.Replace(text, "OpenTable", "CloseTable", 1)
	text += ");\n"
	return text
}

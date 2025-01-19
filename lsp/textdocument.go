package lsp

/**
 * Text Document
 */
type TextDocumentItem struct {
	URI string `json:"uri"`
	LanguageId string `json:"lanuageId"`
	Version int `json:"version"`
	Text string `json:"text"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position Position `json:"position"`
}

type Position struct {
	Line int `json:"line"` 
	Character int `json:"character"`
}

type Location struct {
	URI string `json:"uri"`
	Range Range `json:"range"` 
}

type Range struct {
	Start Position `json:"start"`
	End Position `json:"end"`
}

/**
 * Document Open Notification
 */
type DidOpenTextDocumentNotification struct {
	Notification
	Params DidOpenTextDocumentParams `json:"params"`
}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem
}

/**
 * Document Change Notification
 */
type DidChangeTextDocumentNotification struct {
	Notification
	Params DidChangeTextDocumentParams `json:"params"`
}

type DidChangeTextDocumentParams struct {
	TextDocument VersionedTextDocumentIdentifier `json:"textDocument"`
	ContentChanges []TextDocumentChangeEvent `json:"contentChanges"`
}

type TextDocumentChangeEvent struct {
	Text string `json:"text"`
}

/**
 * Diagnostic Notification
 */
type DiagnosticNotification struct {
	Notification	
	Params DiagnosticParams `json:"params"`
}

type DiagnosticParams struct {
	URI string `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type Diagnostic struct {
	Range Range `json:"range"` 
	Severity int `json:"severity"` 
	Source string `json:"source"` 
	Message string `json:"message"` 
}

/**
 * Hover Request
 */
type HoverRequest struct {
	Request
	Params HoverParams `json:"params"`
}

type HoverParams struct {
	TextDocumentPositionParams
}

type HoverResponse struct {
	Response
	Result HoverResult `json:"result"` 
}

type HoverResult struct {
	Contents string `json:"contents"` 
}

/**
 * Definition Request
 */
type DefinitionRequest struct {
	Request
	Params DefinitionParams `json:"params"`
}

type DefinitionParams struct {
	TextDocumentPositionParams
}

type DefinitionResponse struct {
	Response
	Result Location `json:"result"` 
}

/**
 * Code Action Request
 */
type CodeActionRequest struct {
	Request
	Params CodeActionParams `json:"params"`
}

type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range Range `json:"range"` 
	Context CodeActionContext `json:"context"`
}

type CodeActionResponse struct {
	Response
	Result []CodeAction `json:"result"` 
}

type CodeActionContext struct {
}

type CodeAction struct {
	Title string `json:"title"` 
	Edit *WorkspaceEdit `json:"edit,omitempty"` 
	Command *Command `json:"command,omitempty"` 
}

type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes"`
}

type TextEdit struct {
	Range Range `json:"range"` 
	NewText string `json:"newText"`
}

type Command struct {
	Title string `json:"title"` 
	Command string `json:"command"`
	Argument []interface{} `json:"argument,omitempty"`
}

/**
 * Completion Request
 */
type CompletionRequest struct {
	Request
	Params CompletionParams `json:"params"`
}

type CompletionParams struct {
	TextDocumentPositionParams
}

type CompletionResponse struct {
	Response
	Result []CompletionItem `json:"result"` 
}

type CompletionItem struct {
	Label string `json:"label"` 
	Kind int `json:"kind"` 
	Detail string `json:"detail"` 
	Documentation string `json:"documentation"` 
}

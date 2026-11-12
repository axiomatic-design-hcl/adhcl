package model

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// Language names the exact core contract that a project selects.
type Language string

const LanguageCore1 Language = "requirements_hcl/core/1"

// ExtensionRole is the responsibility that a declared owner fills.
type ExtensionRole string

const (
	ExtensionRoleBinding      ExtensionRole = "binding"
	ExtensionRoleBehaviorLink ExtensionRole = "behavior_link"
	ExtensionRoleView         ExtensionRole = "view"
)

// ListedPath is one path string from a project file list, with its position.
type ListedPath struct {
	Path  string
	Range hcl.Range
}

// Project is the manifest project block. Its label names the input collection
// and does not identify the design.
type Project struct {
	Label           string
	File            string
	Range           hcl.Range
	Language        Language
	CoreFiles       []ListedPath
	AttachmentFiles []ListedPath
}

// Extension is a declared owner. Its ID has the form extension.NAME.
type Extension struct {
	ID       ID
	File     string
	Range    hcl.Range
	Role     ExtensionRole
	Contract string
}

// Attachment is one attachment envelope. Label is unique across the listed
// attachment files.
type Attachment struct {
	Label   string
	File    string
	Range   hcl.Range
	Owner   Ref
	Subject Ref
	Data    []DataField
}

// DataField is one literal key and value from an owner payload, in authored order.
type DataField struct {
	Key        string
	KeyRange   hcl.Range
	Value      cty.Value
	ValueRange hcl.Range
}

// Manifest holds the project and owner declarations of one manifest file.
type Manifest struct {
	Project    *Project
	Extensions []Extension
}

// Attachments holds the attachment envelopes in authored order.
type Attachments []Attachment

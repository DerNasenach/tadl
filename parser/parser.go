// SPDX-FileCopyrightText: © 2021 The tadl authors <https://github.com/golangee/tadl/blob/main/AUTHORS>
// SPDX-License-Identifier: Apache-2.0

package parser

import (
	"fmt"
	"io"

	"github.com/golangee/tadl/token"
)

// TreeNode is a node in the parse tree.
// For regular nodes Text and Comment will always be nil.
// For terminal text nodes Children and Name will be empty and Text will be set.
// For comment nodes Children and Name will be empty and only Comment will be set.
type TreeNode struct {
	Name       string
	Text       *string
	Comment    *string
	Attributes AttributeMap
	parent     *TreeNode
	Children   []*TreeNode
	// BlockType describes the type of brackets the children were surrounded with.
	// This may be BlockNone in which case this node either has no or one children.
	BlockType BlockType
	// Range will span all tokens that were processed to build this node.
	Range token.Position
}

// NewNode creates a new node for the parse tree.
func NewNode(name string) *TreeNode {
	return &TreeNode{
		Name:       name,
		Attributes: NewAttributeMap(),
		BlockType:  BlockNone,
	}
}

// NewTextNode creates a node that will only contain text.
func NewTextNode(cd *token.CharData) *TreeNode {
	return &TreeNode{
		Text: &cd.Value,
		Range: token.Position{
			BeginPos: cd.Begin(),
			EndPos:   cd.End(),
		},
	}
}

// NewCommentNode creates a node that will only contain a comment.
func NewCommentNode(cd *token.CharData) *TreeNode {
	return &TreeNode{
		Comment: &cd.Value,
		Range: token.Position{
			BeginPos: cd.Begin(),
			EndPos:   cd.End(),
		},
	}
}

// NewStringNode will create a text node, like NewTextNode,
// but without positional information. This is only used for testing.
// Use NewTextNode with a CharData token if you can.
func NewStringNode(text string) *TreeNode {
	return &TreeNode{
		Text: &text,
	}
}

// NewStringCommentNode will create a comment node, like NewCommentNode,
// but without positional information. This is only used for testing.
// Use NewCommentNode with a CharData token if you can.
func NewStringCommentNode(text string) *TreeNode {
	return &TreeNode{
		Comment: &text,
	}
}

// AddChildren adds children to a node and can be used builder-style.
func (t *TreeNode) AddChildren(children ...*TreeNode) *TreeNode {
	if t.Children != nil {
		t.Children = append(t.Children, children...)
	} else {
		t.Children = children
	}

	return t
}

// AddAttribute adds an attribute to a node and can be used builder-style.
func (t *TreeNode) AddAttribute(key, value string) *TreeNode {
	t.Attributes.Set(key, value)

	return t
}

// Block is used to set the BlockType of this node.
func (t *TreeNode) Block(blockType BlockType) *TreeNode {
	t.BlockType = blockType

	return t
}

// isClosedBy returns true if tok is a BlockEnd/GroupEnd/GenericEnd that is the correct
// match for closing this TreeNode.
func (t *TreeNode) isClosedBy(tok token.Token) bool {
	switch tok.(type) {
	case *token.BlockEnd:
		return t.BlockType == BlockNormal
	case *token.GroupEnd:
		return t.BlockType == BlockGroup
	case *token.GenericEnd:
		return t.BlockType == BlockGeneric
	default:
		return false
	}
}

// IsText returns true if this node is a text only node.
// Only one of IsText, IsComment, IsNode should be true.
func (t *TreeNode) IsText() bool {
	return t.Text != nil
}

// IsComment returns true if this node is a comment node.
// Only one of IsText, IsComment, IsNode should be true.
func (t *TreeNode) IsComment() bool {
	return t.Comment != nil
}

// IsNode returns true if this is a regular node.
// Only one of IsText, IsComment, IsNode should be true.
func (t *TreeNode) IsNode() bool {
	return !t.IsText() && !t.IsComment()
}

func (t *TreeNode) Print() string {
	text := t.Name
	fmt.Println(t.Name, t.Text)
	for _, child := range t.Children {
		text += child.Print()
	}

	return text
}

// AttributeMap is a custom map[string]string to make the
// handling of attributes easier.
type AttributeMap map[string]string

func NewAttributeMap() AttributeMap {
	return make(map[string]string)
}

// Set sets a key to a value in this map.
func (a AttributeMap) Set(key, value string) {
	a[key] = value
}

// Has returns true if the given key is in the map and false otherwise.
func (a AttributeMap) Has(key string) bool {
	_, ok := a[key]
	return ok
}

// Merge returns a new AttributeMap with all keys from this and the other AttributeMap.
func (a AttributeMap) Merge(other AttributeMap) AttributeMap {
	result := NewAttributeMap()

	for k, v := range a {
		result[k] = v
	}

	for k, v := range other {
		result[k] = v
	}

	return result
}

// tokenWithError is a struct that wraps a Token and an error that may
// have occurred while reading that Token.
// This type simplifies storing tokens in the parser.
type tokenWithError struct {
	tok token.Token
	err error
}

// BlockType is an addition for nodes that describes with what brackets their children were surrounded.
type BlockType string

const (
	BlockNone    BlockType = ""
	BlockNormal  BlockType = "{}"
	BlockGroup   BlockType = "()"
	BlockGeneric BlockType = "<>"
)

// Parser is used to get a tree representation from Tadl input.
type Parser struct {
	// forwardingNodes is a list of all nodes that were defined as forwarded.
	// They will be inserted into the next node.
	forwardingNodes      []*TreeNode
	forwardingAttributes AttributeMap

	// root and parent are pointers to work with the successively built Tree.
	// root holds the root Node, parent holds the currently to modify Node
	root   *TreeNode
	parent *TreeNode



	visitor Visitor

	firstNode bool
}

func NewParser(filename string, r io.Reader) *Parser {
	parser := &Parser{
		visitor: *NewVisitor(nil, token.NewLexer(filename, r)),
		root:    NewNode("root"),
	}
	parser.parent = parser.root
	parser.parent = parser.root
	parser.visitor.SetVisitable(parser)
	parser.firstNode = true
	return parser
}

/*func NewParserEncoder(filename string, r io.Reader) *Parser {
	return &Parser{
		lexer:   NewLexer(filename, r),
		mode:    G1,
		visitor: NewVisitorEncoder(),
	}
}*/

// Parse returns a parsed tree.
func (p *Parser) Parse() (*TreeNode, error) {
	err := p.visitor.Run()
	if err != nil {
		return nil, err
	}

	return p.root, nil
}

func (p *Parser) Open() {
	p.parent = p.parent.Children[len(p.parent.Children)-1]
}

func (p *Parser) NewNode(name string) {
	if p.firstNode {
		p.firstNode = false
		p.parent.Name = name
		return
	}
	p.parent.AddChildren(NewNode(name))
}

func (p *Parser) NewStringNode(name string) {
	p.parent.AddChildren(NewStringNode(name))
}

func (p *Parser) NewTextNode(cd *token.CharData) {
	p.parent.AddChildren(NewTextNode(cd))
}

func (p *Parser) NewCommentNode(cd *token.CharData) {
	p.parent.AddChildren(NewCommentNode(cd))
}

func (p *Parser) NewStringCommentNode(text string) {
	p.parent.AddChildren(NewStringCommentNode(text))
}

func (p *Parser) AddAttribute(key, value string) {
	p.root.Attributes.Set(key, value)
}

func (p *Parser) AddForwardAttribute(m AttributeMap) {
	p.forwardingAttributes.Merge(m)
}

func (p *Parser) Block(blockType BlockType) {
	p.parent.Block(blockType)
}

func (p *Parser) Close() {
	p.parent = p.parent.parent
}

func (p *Parser) AddForwardNode(name string) {
	p.forwardingNodes = append(p.forwardingNodes, NewNode(name))
}

func (p *Parser) AppendForwardingNodes() {
	for _, node := range p.forwardingNodes {
		p.parent.Children = append(p.forwardingNodes, node)
	}
	p.forwardingNodes = nil
}

func (p *Parser) MergeAttributes(m AttributeMap) {
	p.parent.Attributes = p.forwardingAttributes.Merge(m)
}

func (p *Parser) GetForwardingLength() int {
	return len(p.forwardingNodes)
}

func (p *Parser) GetForwardingPosition(i int) token.Node {
	return p.forwardingNodes[i].Range
}

func (p *Parser) SetNodeName(name string) {
	p.parent.Name = name
}

func (p *Parser) SetBlockType(b BlockType) {
	p.root.BlockType = b
}

func (p *Parser) GetBlockType() BlockType {
	return p.root.BlockType
}

func (p *Parser) AppendSubTree() {
	p.parent.AddChildren(p.root)
	p.root = nil
}

func (p *Parser) AppendSubTreeForward() {
	p.forwardingNodes = append(p.forwardingNodes, p.root)
	p.root = nil
}

func (p *Parser) SetEndPos(pos token.Pos) {
	p.parent.Range.EndPos = pos
}

func (p *Parser) InsertForwardNodes(nodes []*TreeNode) {
	p.parent.Children = nodes
}

func (p *Parser) SetNodeText(text string) {
	p.parent.Text = &text
}

func (p *Parser) GetPointerPosition() token.Position {
	return p.parent.Range

}
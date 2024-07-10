package filter

import (
	"sync"
)

type dfaNode struct {
	children map[rune]*dfaNode
	lock     sync.RWMutex
	isLeaf   bool
}

func (n *dfaNode) addChild(r rune) *dfaNode {
	n.lock.Lock()
	defer n.lock.Unlock()

	newChile := newDfaNode()
	n.children[r] = newChile

	return newChile
}

func (n *dfaNode) getChild(r rune) (*dfaNode, bool) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	child, ok := n.children[r]

	return child, ok
}

func (n *dfaNode) delChild(r rune) {
	n.lock.Lock()
	defer n.lock.Unlock()

	delete(n.children, r)
}

func newDfaNode() *dfaNode {
	return &dfaNode{
		children: make(map[rune]*dfaNode),
		isLeaf:   false,
	}
}

type DfaModel struct {
	root *dfaNode
}

func NewDfaModel() *DfaModel {
	return &DfaModel{
		root: newDfaNode(),
	}
}

func (m *DfaModel) AddWords(words ...string) {
	for _, word := range words {
		m.addWord(word)
	}
}

func (m *DfaModel) addWord(word string) {
	now := m.root
	runes := []rune(word)

	for _, r := range runes {
		if next, ok := now.children[r]; ok {
			now = next
		} else {
			now = now.addChild(r)
		}
	}

	now.isLeaf = true
}

func (m *DfaModel) DelWords(words ...string) {
	for _, word := range words {
		m.delWord(word)
	}
}

func (m *DfaModel) delWord(word string) {
	var lastLeaf *dfaNode
	var lastLeafNextRune rune
	now := m.root
	runes := []rune(word)

	for _, r := range runes {
		if next, ok := now.getChild(r); !ok {
			return
		} else {
			if next.isLeaf {
				lastLeaf = now
				lastLeafNextRune = r
			}
			now = next
		}
	}
	if lastLeaf != nil {
		lastLeaf.delChild(lastLeafNextRune)
	}
}

func (m *DfaModel) Listen(addChan, delChan <-chan string) {
	go func() {
		for word := range addChan {
			m.addWord(word)
		}
	}()

	go func() {
		for word := range delChan {
			m.delWord(word)
		}
	}()
}

func (m *DfaModel) FindAll(text string) []string {
	var matches []string // stores words that match in dict
	var found bool       // if current rune in node's map
	var now *dfaNode     // current node

	start := 0
	parent := m.root
	runes := []rune(text)
	length := len(runes)

	for pos := 0; pos < length; pos++ {
		now, found = parent.getChild(runes[pos])

		if !found {
			parent = m.root
			pos = start
			start++
			continue
		}

		if now.isLeaf && start <= pos {
			matches = append(matches, string(runes[start:pos+1]))
		}

		if pos == length-1 {
			parent = m.root
			pos = start
			start++
			continue
		}

		parent = now
	}

	var res []string
	set := make(map[string]struct{})

	for _, word := range matches {
		if _, ok := set[word]; !ok {
			set[word] = struct{}{}
			res = append(res, word)
		}
	}

	return res
}

func (m *DfaModel) FindAllCount(text string) map[string]int {
	res := make(map[string]int)
	var found bool
	var now *dfaNode

	start := 0
	parent := m.root
	runes := []rune(text)
	length := len(runes)

	for pos := 0; pos < length; pos++ {
		now, found = parent.getChild(runes[pos])

		if !found {
			parent = m.root
			pos = start
			start++
			continue
		}

		if now.isLeaf && start <= pos {
			res[string(runes[start:pos+1])]++
		}

		if pos == length-1 {
			parent = m.root
			pos = start
			start++
			continue
		}

		parent = now
	}

	return res
}

func (m *DfaModel) FindOne(text string) string {
	var found bool
	var now *dfaNode

	start := 0
	parent := m.root
	runes := []rune(text)
	length := len(runes)

	for pos := 0; pos < length; pos++ {
		now, found = parent.getChild(runes[pos])

		if !found || (!now.isLeaf && pos == length-1) {
			parent = m.root
			pos = start
			start++
			continue
		}

		if now.isLeaf && start <= pos {
			return string(runes[start : pos+1])
		}

		parent = now
	}

	return ""
}

func (m *DfaModel) IsSensitive(text string) bool {
	return m.FindOne(text) != ""
}

func (m *DfaModel) Replace(text string, repl rune) string {
	var found bool
	var now *dfaNode

	start := 0
	parent := m.root
	runes := []rune(text)
	length := len(runes)

	for pos := 0; pos < length; pos++ {
		now, found = parent.getChild(runes[pos])

		if !found || (!now.isLeaf && pos == length-1) {
			parent = m.root
			pos = start
			start++
			continue
		}

		if now.isLeaf && start <= pos {
			for i := start; i <= pos; i++ {
				runes[i] = repl
			}
		}

		parent = now
	}

	return string(runes)
}

func (m *DfaModel) Remove(text string) string {
	var found bool
	var now *dfaNode

	start := 0 // 从文本的第几个文字开始匹配
	parent := m.root
	runes := []rune(text)
	length := len(runes)
	filtered := make([]rune, 0, length)

	for pos := 0; pos < length; pos++ {
		now, found = parent.getChild(runes[pos])

		if !found || (!now.isLeaf && pos == length-1) {
			filtered = append(filtered, runes[start])
			parent = m.root
			pos = start
			start++
			continue
		}

		if now.isLeaf {
			start = pos + 1
			parent = m.root
		} else {
			parent = now
		}
	}

	filtered = append(filtered, runes[start:]...)

	return string(filtered)
}

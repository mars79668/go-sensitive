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
	root   *dfaNode
	status int
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

func (m *DfaModel) LoadStatus() int {
	return m.status
}

func (m *DfaModel) Listen(addChan, delChan <-chan string) {
	go func() {
		for word := range addChan {
			m.addWord(word)
			m.status = DICT_LOADING
			if len(addChan) == 0 {
				m.status = DICT_LOAD_OVER
			}
		}
	}()

	go func() {
		for word := range delChan {
			m.delWord(word)
			m.status = DICT_LOADING
			if len(addChan) == 0 {
				m.status = DICT_LOAD_OVER
			}
		}
	}()
}

func (m *DfaModel) FindAll(text string) []string {
	allCount := m.FindAllCount(text)
	var res []string
	for word, _ := range allCount {
		res = append(res, word)
	}
	return res
}

func (m *DfaModel) FindAllCount(text string) map[string]int {
	res := make(map[string]int)
	m.search(text,
		func(pos int, word string) bool {
			res[word]++
			return false
		})
	return res
}

func (m *DfaModel) FindOne(text string) string {
	var res string

	m.search(text,
		func(pos int, word string) bool {
			res = word
			return true
		})
	return res
}

func (m *DfaModel) IsSensitive(text string) bool {
	return m.FindOne(text) != ""
}

func (m *DfaModel) Replace(text string, repl rune) string {
	runes := []rune(text)
	m.search(text,
		func(pos int, word string) bool {
			wr := []rune(word)
			for i := pos - len(wr) + 1; i <= pos; i++ {
				runes[i] = repl
			}
			return false
		})
	return string(runes)
}

func (m *DfaModel) Remove(text string) string {
	runes := []rune(text)
	var res []rune
	m.search(text,
		func(pos int, word string) bool {
			wr := []rune(word)
			res = append(runes[:pos-len(wr)+1], runes[pos+1:]...)
			return false
		})
	return string(res)
}

func (m *DfaModel) search(text string, handler searchHandler) {
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
			word := string(runes[start : pos+1])
			isBreak := handler(pos, word)
			if isBreak {
				return
			}
		}

		if pos == length-1 {
			parent = m.root
			pos = start
			start++
			continue
		}

		parent = now
	}
}

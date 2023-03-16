package parse

type KeywordTrie struct {
	children []*KeywordTrie
	runes    []rune
	leaf     string
}

// Count returns the number of unique keywords which have been added to this KeywordTrie.
func (t *KeywordTrie) Count() int {
	if t == nil {
		return 0
	}
	sum := 0
	if t.leaf != "" {
		sum = 1
	}
	for _, child := range t.children {
		sum += child.Count()
	}
	return sum
}

// Contains returns true iff the provided string is contained in this KeywordTrie
func (t *KeywordTrie) Contains(str string) bool {
	return t.Match([]rune(str)) == str
}

// MatchStr is equivalent to
//
//	Match([]rune(str))
func (t *KeywordTrie) MatchStr(str string) string {
	return t.Match([]rune(str))
}

// Match matches the provided rune slice against the maximal keyword found in this KeywordTrie, returning the matched
// keyword.
func (t *KeywordTrie) Match(stream []rune) string {
	if len(stream) == 0 {
		return t.leaf
	}
	for idx, r := range t.runes {
		if r == stream[0] {
			if result := t.children[idx].Match(stream[1:]); result == "" {
				return t.leaf
			} else {
				return result
			}
		}
	}
	return t.leaf
}

// Add adds the provided keyword to this Trie.
func (t *KeywordTrie) Add(keyword string) {
	t.add(keyword, []rune(keyword))
}

func (t *KeywordTrie) add(orig string, keyword []rune) {
	if len(keyword) == 0 {
		t.leaf = orig
		return
	}
	next := keyword[0]
	for idx, r := range t.runes {
		if r == next {
			t.children[idx].add(orig, keyword[1:])
			return
		}
	}
	child := &KeywordTrie{}
	t.runes = append(t.runes, keyword[0])
	t.children = append(t.children, child)
	child.add(orig, keyword[1:])
}

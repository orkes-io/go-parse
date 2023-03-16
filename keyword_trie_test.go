package parse

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKeywordTrie(t *testing.T) {
	keywords := []string{">=", ">", "<=", "<", "==", "!=", "!"}

	trie := &KeywordTrie{}
	for _, keyword := range keywords {
		trie.Add(keyword)
	}

	assert.Equal(t, "<=", trie.MatchStr("<= 3"))

	assert.Equal(t, "<", trie.MatchStr("< 7"))
	assert.Equal(t, "<", trie.MatchStr("<html>"))
	assert.Equal(t, ">=", trie.MatchStr(">= 14 + 2"))
	assert.Equal(t, "!", trie.MatchStr("!(having_it)"))
	assert.Equal(t, "", trie.MatchStr("5 != 6"))
	assert.Equal(t, "!=", trie.MatchStr("!= 6"))

	assert.True(t, trie.Contains("<="))
	assert.True(t, trie.Contains("!="))
	assert.True(t, trie.Contains("!"))

	assert.Equal(t, 7, trie.Count())
}

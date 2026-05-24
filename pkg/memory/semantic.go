package memory

import (
	"math"
	"strings"
	"unicode"
)

// SearchResult is a memory entry with relevance score.
type SearchResult struct {
	Entry Entry   `json:"entry"`
	Score float64 `json:"score"`
}

// Search returns semantically relevant memory entries using TF-IDF cosine similarity.
func (m *Manager) Search(goalID, query string, topK int) []SearchResult {
	entries := m.Recall(goalID)
	if len(entries) == 0 || query == "" {
		return nil
	}
	if topK <= 0 {
		topK = 5
	}

	queryTokens := tokenize(query)
	if len(queryTokens) == 0 {
		return nil
	}

	docTokens := make([]map[string]float64, len(entries))
	df := make(map[string]int)
	for i, e := range entries {
		tokens := tokenize(e.Key + " " + e.Value)
		tf := termFreq(tokens)
		docTokens[i] = tf
		seen := make(map[string]bool)
		for term := range tf {
			if !seen[term] {
				df[term]++
				seen[term] = true
			}
		}
	}

	queryVec := tfIDF(termFreq(queryTokens), df, len(entries))
	var results []SearchResult
	for i, e := range entries {
		docVec := tfIDF(docTokens[i], df, len(entries))
		score := cosineSimilarity(queryVec, docVec)
		if score > 0 {
			results = append(results, SearchResult{Entry: e, Score: score})
		}
	}

	sortResults(results)
	if len(results) > topK {
		results = results[:topK]
	}
	return results
}

// SearchContext builds a prompt context string from semantic search results.
func (m *Manager) SearchContext(goalID, query string, topK int) string {
	results := m.Search(goalID, query, topK)
	if len(results) == 0 {
		return m.SummarizeContext(goalID, topK)
	}
	var b strings.Builder
	b.WriteString("Relevant memory:\n")
	for _, r := range results {
		b.WriteString("- ")
		b.WriteString(r.Entry.Key)
		b.WriteString(": ")
		b.WriteString(r.Entry.Value)
		b.WriteString("\n")
	}
	return b.String()
}

func tokenize(text string) []string {
	text = strings.ToLower(text)
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if len(f) > 1 {
			out = append(out, f)
		}
	}
	return out
}

func termFreq(tokens []string) map[string]float64 {
	tf := make(map[string]float64)
	for _, t := range tokens {
		tf[t]++
	}
	return tf
}

func tfIDF(tf map[string]float64, df map[string]int, docCount int) map[string]float64 {
	vec := make(map[string]float64, len(tf))
	for term, freq := range tf {
		idf := math.Log(1 + float64(docCount)/float64(1+df[term]))
		vec[term] = freq * idf
	}
	return vec
}

func cosineSimilarity(a, b map[string]float64) float64 {
	var dot, normA, normB float64
	for k, v := range a {
		normA += v * v
		if bv, ok := b[k]; ok {
			dot += v * bv
		}
	}
	for _, v := range b {
		normB += v * v
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func sortResults(results []SearchResult) {
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

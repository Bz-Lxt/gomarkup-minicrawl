package index

import (
	"sort"
	"sync"
)

type Posting struct {
	DocID int   `json:"doc_id"`
	Freq  int   `json:"freq"`
	Pos   []int `json:"positions"`
}

type Document struct {
	ID    int
	URL   string
	Title string
	Body  string
}

type Hit struct {
	DocID     int     `json:"doc_id"`
	URL       string  `json:"url"`
	Title     string  `json:"title"`
	Score     float64 `json:"score"`
	Snippet   string  `json:"snippet"`
	Highlight bool    `json:"highlight"`
}

type Inverted struct {
	mu    sync.RWMutex
	terms map[string][]Posting
	docs  map[int]*Document
	tf    map[string]int
}

func New() *Inverted {
	return &Inverted{
		terms: make(map[string][]Posting),
		docs:  make(map[int]*Document),
		tf:    make(map[string]int),
	}
}

func (inv *Inverted) Add(doc Document) {
	titleTokens := Tokenize(doc.Title)
	bodyTokens := Tokenize(doc.Body)
	tokens := make([]string, 0, len(titleTokens)*3+len(bodyTokens))
	for i := 0; i < 3; i++ {
		tokens = append(tokens, titleTokens...)
	}
	tokens = append(tokens, bodyTokens...)

	type acc struct {
		freq int
		pos  []int
	}
	bag := map[string]*acc{}
	for i, t := range tokens {
		a := bag[t]
		if a == nil {
			a = &acc{}
			bag[t] = a
		}
		a.freq++
		if len(a.pos) < 32 {
			a.pos = append(a.pos, i)
		}
	}

	inv.mu.Lock()
	defer inv.mu.Unlock()
	if old, ok := inv.docs[doc.ID]; ok {
		inv.removeLocked(old.ID)
	}
	inv.docs[doc.ID] = &Document{ID: doc.ID, URL: doc.URL, Title: doc.Title, Body: doc.Body}
	for term, a := range bag {
		inv.terms[term] = append(inv.terms[term], Posting{DocID: doc.ID, Freq: a.freq, Pos: a.pos})
		inv.tf[term] += a.freq
	}
}

func (inv *Inverted) Remove(docID int) {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	inv.removeLocked(docID)
}

func (inv *Inverted) removeLocked(docID int) {
	if _, ok := inv.docs[docID]; !ok {
		return
	}
	delete(inv.docs, docID)
	for term, list := range inv.terms {
		dst := list[:0]
		for _, p := range list {
			if p.DocID == docID {
				inv.tf[term] -= p.Freq
				continue
			}
			dst = append(dst, p)
		}
		if len(dst) == 0 {
			delete(inv.terms, term)
			delete(inv.tf, term)
			continue
		}
		inv.terms[term] = dst
	}
}

func (inv *Inverted) Clear() {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	inv.terms = make(map[string][]Posting)
	inv.docs = make(map[int]*Document)
	inv.tf = make(map[string]int)
}

func (inv *Inverted) Stats() (docs, terms int) {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return len(inv.docs), len(inv.terms)
}

func (inv *Inverted) TopKeywords(limit int) []Keyword {
	if limit <= 0 {
		limit = 30
	}
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	out := make([]Keyword, 0, len(inv.tf))
	for t, f := range inv.tf {
		if f <= 0 {
			continue
		}
		out = append(out, Keyword{Term: t, Freq: f})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Freq == out[j].Freq {
			return out[i].Term < out[j].Term
		}
		return out[i].Freq > out[j].Freq
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

type Keyword struct {
	Term string `json:"term"`
	Freq int    `json:"freq"`
}

func (inv *Inverted) Search(query string, highlight bool, limit int) []Hit {
	terms := QueryTerms(query)
	if len(terms) == 0 {
		return nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	counts := map[int]int{}
	scores := map[int]float64{}
	for _, term := range terms {
		list := inv.terms[term]
		for _, p := range list {
			counts[p.DocID]++
			scores[p.DocID] += float64(p.Freq)
		}
	}
	need := len(terms)
	hits := make([]Hit, 0)
	for id, c := range counts {
		if c < need {
			continue
		}
		doc := inv.docs[id]
		if doc == nil {
			continue
		}
		hits = append(hits, Hit{
			DocID:     doc.ID,
			URL:       doc.URL,
			Title:     doc.Title,
			Score:     scores[id],
			Snippet:   Snippet(doc.Body, query, highlight),
			Highlight: highlight,
		})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].URL < hits[j].URL
		}
		return hits[i].Score > hits[j].Score
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	if highlight {
		for i := range hits {
			hits[i].Title = Highlight(hits[i].Title, query)
		}
	}
	return hits
}

// The CLIP text tower wrapper (M3d Task 3): it feeds one query
// string to a Chinese-CLIP text tower and answers the L2-normalized
// text feature the search's semantic arm matches against the image
// embeddings the vision tower wrote -- both towers share the joint
// embedding space of the same upstream checkpoint.
//
// The tokenization follows the BERT-Chinese convention the
// Chinese-CLIP towers were trained with, implemented here from
// scratch over the model's pinned vocabulary file (the manifest's
// second file): lowercase, whitespace and punctuation splitting with
// every CJK codepoint its own word, then greedy longest-match
// WordPiece with "##" continuations, the [CLS]/[SEP] envelope, and
// padding to the tower's declared token context length with the
// matching attention mask and zero token types.
package mlclip

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strings"
	"unicode"

	ort "github.com/yalue/onnxruntime_go"

	"github.com/welpher/lumemor/server/internal/ml"
)

// TextContextLength is the Chinese-CLIP text towers' token context
// length -- the family's public convention, the text-side sibling of
// the vision tower's 224x224 geometry (and the value the default
// manifest's inputSize pins for display). A graph that declares its
// own fixed length wins over this default, exactly the way the image
// wrapper honors a pinned geometry.
const TextContextLength = 52

// The vocabulary's special tokens: the standard BERT-Chinese
// positions of the pinned table.
const (
	tokenPad = "[PAD]"
	tokenUnk = "[UNK]"
	tokenCLS = "[CLS]"
	tokenSEP = "[SEP]"
)

// Tokenizer is one loaded vocabulary: the token-to-id map and its
// reverse. It is immutable once loaded and safe for concurrent use.
type Tokenizer struct {
	ids    map[string]int64
	tokens []string
}

// NewTokenizer reads and parses the vocabulary file: one token per
// line, the line number (zero-based) the token's id -- the layout of
// the pinned Chinese-CLIP vocab.txt.
func NewTokenizer(vocabPath string) (*Tokenizer, error) {
	file, err := os.Open(vocabPath)
	if err != nil {
		return nil, fmt.Errorf("mlclip: open the vocabulary %s: %w", vocabPath, err)
	}
	defer func() { _ = file.Close() }()

	ids := map[string]int64{}
	var tokens []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		token := scanner.Text()
		if _, duplicate := ids[token]; duplicate {
			return nil, fmt.Errorf("mlclip: the vocabulary lists %q twice", token)
		}
		ids[token] = int64(len(tokens))
		tokens = append(tokens, token)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("mlclip: read the vocabulary %s: %w", vocabPath, err)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("mlclip: the vocabulary %s is empty", vocabPath)
	}
	for _, want := range []string{tokenPad, tokenUnk, tokenCLS, tokenSEP} {
		if _, ok := ids[want]; !ok {
			return nil, fmt.Errorf("mlclip: the vocabulary carries no %s token", want)
		}
	}
	return &Tokenizer{ids: ids, tokens: tokens}, nil
}

// id answers the token's vocabulary id (the zero id for an absent
// token; the encoder never consults it for pieces it did not find).
func (t *Tokenizer) id(token string) int64 {
	if id, ok := t.ids[token]; ok {
		return id
	}
	return 0
}

// token answers the id's vocabulary token (the empty string for an
// out-of-range id; the tests' reverse lookups stay in range).
func (t *Tokenizer) token(id int64) string {
	if id < 0 || id >= int64(len(t.tokens)) {
		return ""
	}
	return t.tokens[id]
}

// Encode tokenizes text into the tower's three input rows: the
// [CLS]/[SEP]-wrapped token ids padded to the context length, the
// attention mask (1 over the real tokens, 0 over the padding) and
// the all-zero token types (a single-segment input). Content tokens
// truncate at contextLength-2 so the [SEP] always closes the
// envelope.
func (t *Tokenizer) Encode(text string) (ids, mask, types []int64) {
	pieces := t.basicTokenize(text)
	content := make([]int64, 0, len(pieces))
	for _, piece := range pieces {
		content = append(content, t.wordPiece(piece)...)
	}
	if len(content) > TextContextLength-2 {
		content = content[:TextContextLength-2]
	}

	ids = make([]int64, TextContextLength)
	mask = make([]int64, TextContextLength)
	types = make([]int64, TextContextLength)
	ids[0] = t.ids[tokenCLS]
	n := 1
	for _, id := range content {
		ids[n] = id
		n++
	}
	ids[n] = t.ids[tokenSEP]
	n++
	for i := range n {
		mask[i] = 1
	}
	return ids, mask, types
}

// basicTokenize splits text per the BERT-Chinese convention: case
// folds, whitespace separates words, every CJK codepoint is its own
// one-character word, and punctuation/symbols break words and stand
// alone.
func (t *Tokenizer) basicTokenize(text string) []string {
	var words []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			words = append(words, string(current))
			current = current[:0]
		}
	}
	for _, r := range strings.ToLower(text) {
		switch {
		case unicode.IsSpace(r):
			flush()
		case isCJK(r), isBreakingPunct(r):
			flush()
			words = append(words, string(r))
		default:
			current = append(current, r)
		}
	}
	flush()
	return words
}

// wordPiece splits one word by greedy longest-match-first against
// the vocabulary, with "##" continuation markers inside the word and
// the [UNK] token when a leading run matches nothing.
func (t *Tokenizer) wordPiece(word string) []int64 {
	runes := []rune(word)
	if len(runes) == 0 {
		return nil
	}
	var pieces []int64
	start := 0
	for start < len(runes) {
		end := len(runes)
		matched := int64(-1)
		for start < end {
			candidate := string(runes[start:end])
			if start > 0 {
				candidate = "##" + candidate
			}
			if id, ok := t.ids[candidate]; ok {
				matched = id
				break
			}
			end--
		}
		if matched < 0 {
			pieces = append(pieces, t.ids[tokenUnk])
			start++
			continue
		}
		pieces = append(pieces, matched)
		start = end
	}
	return pieces
}

// isCJK reports whether the codepoint belongs to the CJK script
// blocks the Chinese-CLIP corpora tokenize per character: the CJK
// unified ideographs (base and extensions), the kana and the
// hangul syllables.
func isCJK(r rune) bool {
	switch {
	case r >= 0x3040 && r <= 0x30FF, // Hiragana + Katakana
		r >= 0x3400 && r <= 0x4DBF,   // CJK ext A
		r >= 0x4E00 && r <= 0x9FFF,   // CJK unified
		r >= 0xAC00 && r <= 0xD7AF,   // Hangul syllables
		r >= 0x20000 && r <= 0x2A6DF: // CJK ext B
		return true
	}
	return false
}

// isBreakingPunct reports whether the codepoint breaks a word: the
// BERT tokenizer's punctuation rule (any punctuation or symbol
// stands alone), which keeps filenames and mixed text honest.
func isBreakingPunct(r rune) bool {
	return unicode.IsPunct(r) || unicode.IsSymbol(r)
}

// EmbedText feeds one query string through the text tower and
// answers its text feature, L2-normalized (the wrapper's contract is
// the unit vector, so a graph whose raw output drifts from the
// family's claim still answers a comparable direction). The session
// must carry the tower's exact input contract: the three named int64
// tensors (input_ids, attention_mask, token_type_ids) of shape
// [batch, tokens] -- a fixed token extent wins over the context
// default -- and one [batch, dim] float output.
func EmbedText(sess *ml.Session, tok *Tokenizer, text string) ([]float32, error) {
	if tok == nil {
		return nil, fmt.Errorf("mlclip: the text tower needs a tokenizer")
	}
	length, err := textInputGeometry(sess)
	if err != nil {
		return nil, err
	}
	if length != TextContextLength {
		// A graph that pins its own token extent is honored the way
		// the image wrapper honors a pinned geometry: the envelope
		// re-encodes at the declared length.
		return embedTextAt(sess, tok, text, length)
	}

	ids, mask, types := tok.Encode(text)
	feature, err := runTextTower(sess, ids, mask, types)
	if err != nil {
		return nil, err
	}
	return normalizeFeature(feature)
}

// embedTextAt encodes at a graph-declared context length: the same
// envelope with the padding adjusted.
func embedTextAt(sess *ml.Session, tok *Tokenizer, text string, length int) ([]float32, error) {
	pieces := tok.basicTokenize(text)
	content := make([]int64, 0, len(pieces))
	for _, piece := range pieces {
		content = append(content, tok.wordPiece(piece)...)
	}
	if len(content) > length-2 {
		content = content[:length-2]
	}
	ids := make([]int64, length)
	mask := make([]int64, length)
	types := make([]int64, length)
	ids[0] = tok.ids[tokenCLS]
	n := 1
	for _, id := range content {
		ids[n] = id
		n++
	}
	ids[n] = tok.ids[tokenSEP]
	n++
	for i := range n {
		mask[i] = 1
	}
	feature, err := runTextTower(sess, ids, mask, types)
	if err != nil {
		return nil, err
	}
	return normalizeFeature(feature)
}

// textInputGeometry validates the tower's input contract and answers
// the token extent to encode at: a pinned extent from the graph, the
// family default when the graph leaves it dynamic.
func textInputGeometry(sess *ml.Session) (int, error) {
	inputs := sess.Inputs()
	if len(inputs) != 3 {
		return 0, fmt.Errorf("mlclip: the text tower carries %d inputs, want 3", len(inputs))
	}
	want := []string{"input_ids", "attention_mask", "token_type_ids"}
	for i, name := range want {
		if inputs[i].Name != name {
			return 0, fmt.Errorf("mlclip: the text tower input %d is %q, want %q", i, inputs[i].Name, name)
		}
		if inputs[i].DataType != ort.TensorElementDataTypeInt64 {
			return 0, fmt.Errorf("mlclip: the text tower input %q is not an int64 tensor", name)
		}
		if len(inputs[i].Dimensions) != 2 {
			return 0, fmt.Errorf("mlclip: the text tower input %q has %d dimensions, want [batch, tokens]",
				name, len(inputs[i].Dimensions))
		}
		if inputs[i].Dimensions[0] != -1 && inputs[i].Dimensions[0] != 1 {
			return 0, fmt.Errorf("mlclip: the text tower input %q batch is %d, want 1 or dynamic",
				name, inputs[i].Dimensions[0])
		}
	}
	if outputs := sess.Outputs(); len(outputs) != 1 {
		return 0, fmt.Errorf("mlclip: the text tower carries %d outputs, want 1", len(outputs))
	}
	if extent := inputs[0].Dimensions[1]; extent > 0 {
		return int(extent), nil
	}
	return TextContextLength, nil
}

// runTextTower feeds the three rows through the graph and answers
// the raw feature of the single batch entry.
func runTextTower(sess *ml.Session, ids, mask, types []int64) ([]float32, error) {
	shape := ort.NewShape(1, int64(len(ids)))
	idsTensor, err := ort.NewTensor(shape, ids)
	if err != nil {
		return nil, fmt.Errorf("mlclip: build the input_ids tensor: %w", err)
	}
	defer func() { _ = idsTensor.Destroy() }()
	maskTensor, err := ort.NewTensor(shape, mask)
	if err != nil {
		return nil, fmt.Errorf("mlclip: build the attention_mask tensor: %w", err)
	}
	defer func() { _ = maskTensor.Destroy() }()
	typesTensor, err := ort.NewTensor(shape, types)
	if err != nil {
		return nil, fmt.Errorf("mlclip: build the token_type_ids tensor: %w", err)
	}
	defer func() { _ = typesTensor.Destroy() }()

	outputs := make([]ort.Value, 1)
	if err := sess.Run([]ort.Value{idsTensor, maskTensor, typesTensor}, outputs); err != nil {
		return nil, fmt.Errorf("mlclip: run the text tower: %w", err)
	}
	defer func() { _ = outputs[0].Destroy() }()

	name := sess.Outputs()[0].Name
	tensor, ok := outputs[0].(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("mlclip: output %q is %T, want a float32 tensor", name, outputs[0])
	}
	feature := tensor.GetData()
	if len(feature) == 0 {
		return nil, fmt.Errorf("mlclip: the text tower %s emitted an empty feature", name)
	}
	return feature, nil
}

// normalizeFeature L2-normalizes the tower's raw output into the
// wrapper's unit-vector contract.
func normalizeFeature(feature []float32) ([]float32, error) {
	norm := 0.0
	for _, component := range feature {
		norm += float64(component) * float64(component)
	}
	if norm == 0 {
		return nil, fmt.Errorf("mlclip: the text tower emitted a zero vector; it cannot be normalized")
	}
	scale := float32(1 / math.Sqrt(norm))
	unit := make([]float32, len(feature))
	for i, component := range feature {
		unit[i] = component * scale
	}
	return unit, nil
}

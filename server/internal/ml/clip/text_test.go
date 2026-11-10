// The CLIP text tower wrapper tests: the tokenizer's
// contract over the committed real vocabulary (the Chinese-CLIP BERT
// WordPiece table) and the EmbedText wiring over the self-made
// text-tower fixture graph. The fixture maps token ids to a
// deterministic feature vector, so the tests pin the exact plumbing:
// the [CLS]/[SEP] envelope, the padding, the attention mask, the
// token_type zeros, the fixed context length, the batch-one shape and
// the L2 normalization -- plus the real-model smoke (env-gated) over
// the downloaded text tower.
package mlclip

import (
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	ort "github.com/yalue/onnxruntime_go"

	"github.com/welpher/lumemor/server/internal/ml"
)

// textEmbedFixtureSHA256 pins the self-made text-tower fixture (the
// digest of tools/models/cache/fixtures/text_embed.onnx, committed
// here as testdata by this task).
const textEmbedFixtureSHA256 = "0acd1246d21ac8dda64a7de1a6e830161554a68bd6b1380e6c239f7ebc588499"

// vocabPath is the committed real tokenizer vocabulary: the
// Chinese-CLIP/BERT-Chinese table (byte-identical to the canonical
// vocab.txt of OFA-Sys/chinese-clip-vit-base-patch16, the table the
// default manifest's text tower pins for download).
const vocabPath = "../testdata/cn_clip_vocab.txt"

// loadTextFixture opens the text-tower fixture session.
func loadTextFixture(t *testing.T) *ml.Session {
	t.Helper()
	if !ml.RuntimeAvailable() {
		t.Skip("onnxruntime shared library not present")
	}
	sess, err := ml.NewSession(filepath.Join("..", "testdata", "text_embed.onnx"), textEmbedFixtureSHA256)
	if err != nil {
		t.Fatalf("NewSession(text fixture) error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = sess.Close() })
	return sess
}

// loadRealVocabulary loads the committed vocabulary.
func loadRealVocabulary(t *testing.T) *Tokenizer {
	t.Helper()
	tok, err := NewTokenizer(vocabPath)
	if err != nil {
		t.Fatalf("NewTokenizer(%q) error = %v, want nil", vocabPath, err)
	}
	return tok
}

// wantSpecialTokens pins the vocabulary's special-token ids: the
// standard BERT-Chinese positions ([PAD] 0, [UNK] 100, [CLS] 101,
// [SEP] 102) of the committed table.
func wantSpecialTokens() map[string]int64 {
	return map[string]int64{"[PAD]": 0, "[UNK]": 100, "[CLS]": 101, "[SEP]": 102}
}

// TestTokenizerSpecialTokenIds pins the special-token ids the
// envelope relies on: they are properties of the committed vocabulary
// file, not of the tokenizer code.
func TestTokenizerSpecialTokenIds(t *testing.T) {
	tok := loadRealVocabulary(t)
	for token, want := range wantSpecialTokens() {
		if got := tok.id(token); got != want {
			t.Errorf("id(%q) = %d, want %d", token, got, want)
		}
	}
}

// TestTokenizerEncodePinsTheEnvelope pins the encoding of the text
// arm's representative inputs: the [CLS]/[SEP] envelope, CJK characters
// as single-character words, Latin WordPiece splitting with ##
// continuations, lowercasing, the fixed context length, the padding
// and the attention mask. The expectations were derived from the
// committed vocabulary (the tokenizer's own table lookups), so a
// drifted vocabulary file fails here loudly.
func TestTokenizerEncodePinsTheEnvelope(t *testing.T) {
	tok := loadRealVocabulary(t)

	// The token STRING sequence is the pinned expectation; the ids
	// follow from the vocabulary through the reverse table.
	cases := []struct {
		name string
		text string
		toks []string
	}{
		{
			name: "ChinesePair",
			text: "海边",
			toks: []string{"[CLS]", "海", "边", "[SEP]"},
		},
		{
			name: "EnglishWordPiece",
			text: "Beaches",
			toks: []string{"[CLS]", "beach", "##es", "[SEP]"},
		},
		{
			name: "MixedScript",
			text: "海边 Beach",
			toks: []string{"[CLS]", "海", "边", "beach", "[SEP]"},
		},
		{
			name: "PunctuationSplits",
			text: "海,边",
			toks: []string{"[CLS]", "海", ",", "边", "[SEP]"},
		},
		{
			name: "UnknownIsUNK",
			text: "\U0001f600",
			toks: []string{"[CLS]", "[UNK]", "[SEP]"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ids, mask, types := tok.Encode(tc.text)
			if len(ids) != TextContextLength || len(mask) != TextContextLength || len(types) != TextContextLength {
				t.Fatalf("encode lengths = %d/%d/%d, want all %d", len(ids), len(mask), len(types), TextContextLength)
			}
			got := make([]string, 0, len(tc.toks))
			ones := 0
			for i, id := range ids {
				if mask[i] != 0 && mask[i] != 1 {
					t.Errorf("mask[%d] = %d, want 0 or 1", i, mask[i])
				}
				if mask[i] == 1 {
					ones++
					got = append(got, tok.token(id))
				} else if id != 0 {
					t.Errorf("ids[%d] = %d behind the mask, want [PAD] 0", i, id)
				}
			}
			if !slices.Equal(got, tc.toks) {
				t.Errorf("tokens = %v, want %v", got, tc.toks)
			}
			if ones != len(tc.toks) {
				t.Errorf("masked-on count = %d, want %d", ones, len(tc.toks))
			}
			for _, id := range types {
				if id != 0 {
					t.Errorf("token_type id = %d, want 0 (single-segment input)", id)
				}
			}
		})
	}
}

// TestTokenizerTruncatesAtContextLength pins the truncation: content
// tokens stop at TextContextLength-2 so the [SEP] always closes the
// envelope.
func TestTokenizerTruncatesAtContextLength(t *testing.T) {
	tok := loadRealVocabulary(t)

	text := strings.Repeat("海", TextContextLength+10)
	ids, mask, _ := tok.Encode(text)
	content := 0
	for _, m := range mask {
		if m == 1 {
			content++
		}
	}
	// [CLS] + (context-2) content + [SEP] = context
	if content != TextContextLength {
		t.Errorf("masked-on count = %d, want %d (the full envelope, truncated content)", content, TextContextLength)
	}
	if ids[TextContextLength-1] != wantSpecialTokens()["[SEP]"] {
		t.Errorf("last id = %d, want the [SEP] %d", ids[TextContextLength-1], wantSpecialTokens()["[SEP]"])
	}
}

// TestEmbedTextWiring pins the fixture wiring: the output dimension,
// the L2 normalization, and the deterministic dependence on the token
// ids -- the same text embeds identically, a different text embeds a
// different direction.
func TestEmbedTextWiring(t *testing.T) {
	sess := loadTextFixture(t)
	tok := loadRealVocabulary(t)

	first, err := EmbedText(sess, tok, "海边")
	if err != nil {
		t.Fatalf("EmbedText(海边) error = %v, want nil", err)
	}
	if len(first) != 512 {
		t.Fatalf("embedding dimension = %d, want 512", len(first))
	}
	norm := 0.0
	for _, c := range first {
		norm += float64(c) * float64(c)
	}
	if diff := math.Abs(norm - 1); diff > 1e-4 {
		t.Errorf("L2 norm = %v, want 1", norm)
	}

	again, err := EmbedText(sess, tok, "海边")
	if err != nil {
		t.Fatalf("EmbedText(海边) again error = %v, want nil", err)
	}
	if !slices.Equal(first, again) {
		t.Error("the same text embedded twice differs, want the identical vector")
	}

	other, err := EmbedText(sess, tok, "山脉")
	if err != nil {
		t.Fatalf("EmbedText(山脉) error = %v, want nil", err)
	}
	if slices.Equal(first, other) {
		t.Error("different texts embed to the identical vector, want different directions")
	}
}

// TestEmbedTextSessionContract pins the session input contract: the
// graph must carry exactly the three named int64 inputs
// (input_ids/attention_mask/token_type_ids) with the two-dimensional
// [batch, tokens] shape; the identity and vision fixtures fail the
// refusal paths.
func TestEmbedTextSessionContract(t *testing.T) {
	tok := loadRealVocabulary(t)

	// The identity graph: one float input -- the wrong contract.
	identity, err := ml.NewSession(filepath.Join("..", "testdata", "identity.onnx"), identityFixtureSHA256)
	if err != nil {
		t.Fatalf("NewSession(identity) error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = identity.Close() })
	if _, err := EmbedText(identity, tok, "海边"); err == nil {
		t.Error("EmbedText over the identity graph error = nil, want the input-contract refusal")
	}

	// The const512 vision graph: one float image input -- also wrong.
	vision, err := ml.NewSession(filepath.Join("..", "testdata", "const512.onnx"), const512FixtureSHA256)
	if err != nil {
		t.Fatalf("NewSession(const512) error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = vision.Close() })
	if _, err := EmbedText(vision, tok, "海边"); err == nil {
		t.Error("EmbedText over the vision graph error = nil, want the input-contract refusal")
	}

	// The text fixture itself: the metadata must be the pinned
	// three-input convention.
	sess := loadTextFixture(t)
	inputs := sess.Inputs()
	if len(inputs) != 3 {
		t.Fatalf("session reports %d inputs, want 3", len(inputs))
	}
	wantNames := []string{"input_ids", "attention_mask", "token_type_ids"}
	for i, want := range wantNames {
		if inputs[i].Name != want {
			t.Errorf("input[%d] name = %q, want %q", i, inputs[i].Name, want)
		}
		if inputs[i].DataType != ort.TensorElementDataTypeInt64 {
			t.Errorf("input[%d] type = %v, want int64", i, inputs[i].DataType)
		}
		if len(inputs[i].Dimensions) != 2 {
			t.Errorf("input[%d] shape = %v, want [batch, tokens]", i, inputs[i].Dimensions)
		}
	}
	if outputs := sess.Outputs(); len(outputs) != 1 || outputs[0].Dimensions[1] != 512 {
		t.Errorf("outputs = %v, want one [batch, 512] feature", outputs)
	}
}

// TestTokenizerVocabularyRoundTrip pins the vocabulary's own shape:
// 21128 entries (the Chinese-CLIP BERT-Chinese table size) and every
// id resolving back to its token.
func TestTokenizerVocabularyRoundTrip(t *testing.T) {
	tok := loadRealVocabulary(t)
	if got := len(tok.ids); got != 21128 {
		t.Errorf("vocabulary size = %d, want 21128", got)
	}
	sample := []string{"海", "边", "beach", "##es", "[CLS]", "[SEP]", "[PAD]", "[UNK]"}
	for _, token := range sample {
		if id, ok := tok.ids[token]; !ok || tok.token(id) != token {
			t.Errorf("vocabulary round trip of %q failed (id lookup %+v)", token, tok.ids[token])
		}
	}
}

// textTowerWeightsSHA256 pins the downloaded text tower (the digest
// the default manifest also pins).
const textTowerWeightsSHA256 = "6d8de6ba498ae2c944ec2b578371c1b3e9dd5a9a991dd4f7dabc16b8111c6f4d"

// TestEmbedTextRealWeights is the optional real-model smoke run:
// LUMEMOR_TEST_CNCLIP_TEXT points at the downloaded text tower
// (tools/models/cache/zihuv_text.onnx in a checkout), and the test
// pins the measured graph shape (the three named int64 inputs with a
// dynamic token extent, the [batch, 512] feature output) and the
// embedding contract: 512 dimensions, unit norm, the same text stable
// across two runs.
func TestEmbedTextRealWeights(t *testing.T) {
	if !ml.RuntimeAvailable() {
		t.Skip("onnxruntime shared library not present")
	}
	path := os.Getenv("LUMEMOR_TEST_CNCLIP_TEXT")
	if path == "" {
		t.Skip("LUMEMOR_TEST_CNCLIP_TEXT not set; point it at the downloaded CLIP text tower to run the real-model smoke")
	}

	sess, err := ml.NewSession(path, textTowerWeightsSHA256)
	if err != nil {
		t.Fatalf("NewSession(cnclip text) error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = sess.Close() })

	inputs := sess.Inputs()
	if len(inputs) != 3 {
		t.Fatalf("session reports %d inputs, want 3", len(inputs))
	}
	wantNames := []string{"input_ids", "attention_mask", "token_type_ids"}
	for i, want := range wantNames {
		if inputs[i].Name != want {
			t.Errorf("input[%d] name = %q, want %q", i, inputs[i].Name, want)
		}
		if len(inputs[i].Dimensions) != 2 {
			t.Errorf("input[%d] shape = %v, want [batch, tokens]", i, inputs[i].Dimensions)
		}
	}
	if outputs := sess.Outputs(); len(outputs) != 1 || outputs[0].Dimensions[1] != 512 {
		t.Fatalf("outputs = %v, want one [batch, 512] feature", outputs)
	}

	tok := loadRealVocabulary(t)
	first, err := EmbedText(sess, tok, "海边")
	if err != nil {
		t.Fatalf("EmbedText error = %v, want nil", err)
	}
	if len(first) != 512 {
		t.Fatalf("embedding dimension = %d, want 512", len(first))
	}
	norm := 0.0
	for _, c := range first {
		norm += float64(c) * float64(c)
	}
	if diff := math.Abs(norm - 1); diff > 1e-4 {
		t.Errorf("L2 norm = %v, want 1", norm)
	}
	second, err := EmbedText(sess, tok, "海边")
	if err != nil {
		t.Fatalf("EmbedText again error = %v, want nil", err)
	}
	if !slices.Equal(first, second) {
		t.Error("the same text embedded twice differs, want the identical vector")
	}
}

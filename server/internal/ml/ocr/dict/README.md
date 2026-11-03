# The OCR recognition dictionary

`ppocr_keys_v1.txt` is the character dictionary of the PP-OCR
recognition family (the Chinese-English model of the default group,
`ppocr-v4-mobile-ch`): one character per line, 6623 lines. It maps the
recognition graph's class indices onto characters -- class 0 is the CTC
blank, classes 1..6623 are these lines in order, class 6624 is the
space (the family's `use_space_char` convention; the dictionary file
itself carries no space line). The recognition graph's output width
(6625) is exactly this table's size, and the wrapper refuses a graph
whose class count disagrees with it.

The file is a verbatim byte-for-byte copy of the upstream PaddleOCR
dictionary:

- Source: https://raw.githubusercontent.com/PaddlePaddle/PaddleOCR/main/ppocr/utils/ppocr_keys_v1.txt
  (the upstream repository moved the file out of `ppocr/utils/dict/`;
  it lives directly under `ppocr/utils/` on the main branch)
- Upstream project: PaddlePaddle/PaddleOCR
- License: Apache-2.0 (the PaddleOCR repository's license)
- Local sha256: `a1c84d9bdb9ab29043c58896224d32941783eb821629618416dcb08f12886492`
  (6623 lines, 26250 bytes, UTF-8)

The RapidOCR Hugging Face repository that hosts the default group's
detection and recognition weights does not carry the dictionary (Task
1's deferred finding), so the dictionary is vendored here from the
PaddleOCR upstream and embedded into the binary via `go:embed`. It is
the only third-party content in this directory; keep the copy
byte-identical to upstream and update the sha256 above whenever it is
refreshed.

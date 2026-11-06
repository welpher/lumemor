# ml testdata provenance

| File | Origin | License / terms |
| --- | --- | --- |
| `identity.onnx` | Self-made fixture (`tools/model-export/make_fixture_onnx.py`): echoes a rank-1 float tensor. | None (hand-built graph, no third-party weights). |
| `const128.onnx`, `const512.onnx`, `const768.onnx` | Self-made fixtures (`tools/model-export/make_fixture_onnx.py`): constant-ones embedding outputs of the given dimension. | None (hand-built graphs, no third-party weights). |
| `ocr_det.onnx` | Self-made fixture (`tools/model-export/make_fixture_onnx.py`, Task 8): fixed 1x3x48x64 input answered by a constant probability map (three 0.9 words over two rows plus one 0.4 region the score filter drops). | None (hand-built graph, no third-party weights). |
| `ocr_rec.onnx` | Self-made fixture (`tools/model-export/make_fixture_onnx.py`, Task 8): any NCHW raster answered by a fixed one-hot CTC sequence `[1, 10, 6625]` decoding to "OCR" (blank/repeat/collapse rules all exercised). | None (hand-built graph, no third-party weights). |
| `text_embed.onnx` | Self-made fixture (`tools/model-export/make_fixture_onnx.py`, M3d Task 3): the CLIP text-tower stand-in with the real tower's input/output contract -- three named int64 inputs (input_ids/attention_mask/token_type_ids) of dynamic [batch, tokens] shape answered by a deterministic [batch, 512] feature (token id + 1 gathered at the context wrap), deliberately non-unit so the wrapper's normalization is observable. | None (hand-built graph, no third-party weights). |
| `cn_clip_vocab.txt` | The canonical Chinese-CLIP tokenizer vocabulary, byte-identical to `OFA-Sys/chinese-clip-vit-base-patch16`'s `vocab.txt` (the same bytes the default manifest's cnclip-vit-b16-text model downloads; digests compared at pin time). 21,128 lines of plain text -- a tokenizer table, not weights. | The vocabulary's own terms: the Chinese-CLIP project's BERT-Chinese table. Redistribution is the table's normal use; the upstream weights card declares no license (the NOTICES release-check item covers the towers; the vocabulary itself is a standard BERT-Chinese table). |
| `ocr-text.png` | Self-made rendering (Task 8): the lines "OCR 2026" and "LUMEMOR M3A" drawn once with the freely-licensed DejaVu Sans font (Bitstream Vera license, permitted for rendering and redistribution) into a 760x300 white canvas; the rasterized image is committed, the font file is not. The real-model OCR smoke's text-bearing input. | The image is self-made content; the glyph shapes originate from DejaVu Sans (free license, attribution preserved here). |
| `face_detection_yunet_2023mar.onnx` | Copied from the OpenCV Zoo release (`opencv/opencv_zoo`, `models/face_detection_yunet/`), the upstream artifact pinned by `manifests/default.json`. The only third-party weight committed to the repository; redistribution is permitted by the MIT license below. | MIT, Copyright (c) 2020 Shiqi Yu (the model directory's LICENSE file, reproduced below). |
| `face-portrait.jpg` | 500px rendition of a public-domain NASA portrait (Kalpana Chawla, official astronaut portrait), fetched from Wikimedia Commons (`File:Kalpana Chawla, NASA photo portrait in orange suit.jpg`), re-encoded as JPEG quality 75. | Public domain (work of NASA). Committed as the face-bearing image for the detection job tests (Task 6); the repository carries no other portrait fixture. |
| `face-portrait-orientation6.jpg` | Derived from `face-portrait.jpg` (Task 6): the upright frame rotated a quarter turn and re-encoded as JPEG quality 90 with EXIF orientation 6, so the stored sensor frame is 625x500 and a decoder honoring the tag restores the upright 500x625 frame. Generated once with libvips; the generation round-trip (reload, AutoRotate, compare) was verified before committing. | Public domain (work of NASA), as the derivative of the portrait above. Pins the detection job's EXIF-orientation handling: the face must land where the upright fixture's face lands. |
| `face-portrait-other.jpg` | 625px rendition of a public-domain NASA portrait of a DIFFERENT person (Richard Husband, the STS-107 commander photographed in the same orange-suit portrait series as the portrait above), fetched from Wikimedia Commons (`File:Richard Husband, NASA photo portrait in orange suit.jpg`), re-encoded as JPEG quality 75. | Public domain (work of NASA). Committed as the first photo of the second person of the people acceptance (M3b): the cross-person cluster-separation evidence needs a face the recognizer cannot group with the portrait above. |
| `face-portrait-other-second.jpg` | A second public-domain NASA photograph of the SAME second person (Richard Husband, a 384x480 portrait), fetched from Wikimedia Commons (`File:Rick Husband.jpg`), re-encoded as JPEG quality 75. | Public domain (work of NASA). Committed as the second photo of the second person: the clusterer's density rule (minPts 2) means one face alone is noise with no cluster id, so the people acceptance's two-cluster structure -- the second person's own cluster, separated from the first person's -- needs two photos of the second person, exactly as the first person's pair needs its two. |

YuNet license (from `models/face_detection_yunet/LICENSE` of the OpenCV Zoo
repository), retained with the redistributed weight:

```text
MIT License

Copyright (c) 2020 Shiqi Yu <shiqi.yu@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

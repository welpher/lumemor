# Model Notices

Provenance and license record for the third-party model artifacts
referenced by the default model manifest
(`server/internal/ml/manifests/default.json`). This file is the
release-check list carrier: anything marked **unresolved** below must
be verified before a public release.

The repository distributes no model weights. Every manifest entry is
a pure link: upstream URL + sha256 + byte count. Downloads happen at
run time (first-boot model download), optionally through an HF domain
mirror, so nothing is re-hosted or re-distributed by this project.
The only committed third-party weight, per project policy, is the
~232 KB YuNet detection model copied into test data by a later task.
All license facts below were verified against the upstream sources on
2026-11-03.

## Pinned artifacts

| Model | File | sha256 | Bytes |
| --- | --- | --- | --- |
| yunet-2023mar | face_detection_yunet_2023mar.onnx | `8f2383e4dd3cfbb4553ea8718107fc0423210dc964f9f4280604804ed2552fa4` | 232,589 |
| sface-2021dec | face_recognition_sface_2021dec.onnx | `0ba9fbfa01b5270c96627c4ef784da859931e02f04419c829e83484087c34e79` | 38,696,353 |
| cnclip-vit-b16-vision | cn_clip_vision.onnx | `980020bf226e528a202de26c6a6186df125b74c612b09cfcd797a082e046b628` | 345,839,119 |
| ppocr-v4-mobile-ch | ch_PP-OCRv4_det_infer.onnx | `d2a7720d45a54257208b1e13e36a8479894cb74155a5efe29462512d42f49da9` | 4,745,517 |
| ppocr-v4-mobile-ch | ch_PP-OCRv4_rec_infer.onnx | `48fc40f24f6d2a207a2b1091d3437eb3cc3eb6b676dc3ef9c37384005483683b` | 10,857,958 |

## yunet-2023mar — face detection

- Source: <https://huggingface.co/opencv/opencv_zoo/resolve/main/models/face_detection_yunet/face_detection_yunet_2023mar.onnx>
  (fallback: the same path on GitHub,
  `https://media.githubusercontent.com/media/opencv/opencv_zoo/main/...`).
- Attribution: OpenCV Zoo (`opencv/opencv_zoo`), which repackages the
  YuNet model by Shiqi Yu (originally libfacedetection).
- License: **MIT** — the repo ships a per-model license file at
  `models/face_detection_yunet/LICENSE`: "MIT License, Copyright (c)
  2020 Shiqi Yu <shiqi.yu@gmail.com>". Commercial use: permitted.

## sface-2021dec — face recognition

- Source: <https://huggingface.co/opencv/opencv_zoo/resolve/main/models/face_recognition_sface/face_recognition_sface_2021dec.onnx>
  (fallback: the same path on GitHub, same scheme as YuNet).
- Attribution: OpenCV Zoo (`opencv/opencv_zoo`), which repackages the
  SFace model from the paper "SFace: An Efficient and Robust Face
  Recognition Network" (Wang et al.).
- License: **Apache-2.0** — the repo ships a per-model license file at
  `models/face_recognition_sface/LICENSE` (Apache License 2.0).
  Commercial use: permitted.

## cnclip-vit-b16-vision — CLIP image embedding

- Source: <https://huggingface.co/felixdu/chinese-clip-vit-base-patch16-onnx/resolve/main/cn_clip_vision.onnx>.
  A community ONNX conversion of the Chinese-CLIP ViT-B/16 vision
  tower (weights: `OFA-Sys/chinese-clip-vit-base-patch16`; conversion
  repo: `felixdu/chinese-clip-vit-base-patch16-onnx`). No natural
  second source exists, so the manifest sets `fallbackUrl` to null.
- License: **unresolved — release-check item.** Two gaps, recorded
  honestly:
  1. The upstream weight card
     (`OFA-Sys/chinese-clip-vit-base-patch16`) carries no license
     field (verified via the HF model API on 2026-11-03).
  2. The conversion repo
     (`felixdu/chinese-clip-vit-base-patch16-onnx`) has no license
     tag and no license declaration in its README (checked on
     2026-11-03).
- Trust semantics: the artifact is pinned by sha256 (upstream drift is
  rejected at download and load time) and the source is accepted
  one-time. Before any public release, confirm the licensing of the
  original Chinese-CLIP weights with the authors or replace this
  entry with a cleanly licensed conversion.

## ppocr-v4-mobile-ch — OCR

- Sources:
  <https://huggingface.co/SWHL/RapidOCR/resolve/main/PP-OCRv4/ch_PP-OCRv4_det_infer.onnx>
  and
  <https://huggingface.co/SWHL/RapidOCR/resolve/main/PP-OCRv4/ch_PP-OCRv4_rec_infer.onnx>.
- Attribution: the PP-OCRv4 mobile Chinese detection and recognition
  networks from the PaddleOCR project (PaddlePaddle), hosted as ONNX
  by the RapidOCR project (`SWHL/RapidOCR` — "hosting ONNX models
  converted from the PaddleOCR project", PPOCR v1–v4). No natural
  second source is pinned; the manifest sets `fallbackUrl` to null.
- License: **Apache-2.0** — the hosting repo's model card declares
  `license: apache-2.0` (README frontmatter, checked 2026-11-03), and
  the upstream PaddleOCR/PaddlePaddle models are Apache-2.0.
  Commercial use: permitted.

## Local cache

One-time downloads for checksum computation and offline testing live
under `tools/models/cache/` (git-ignored). Nothing there is committed;
the manifest links above are the only distribution channel.

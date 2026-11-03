# ml testdata provenance

| File | Origin | License / terms |
| --- | --- | --- |
| `identity.onnx` | Self-made fixture (`tools/model-export/make_fixture_onnx.py`): echoes a rank-1 float tensor. | None (hand-built graph, no third-party weights). |
| `const128.onnx`, `const512.onnx`, `const768.onnx` | Self-made fixtures (`tools/model-export/make_fixture_onnx.py`): constant-ones embedding outputs of the given dimension. | None (hand-built graphs, no third-party weights). |
| `face_detection_yunet_2023mar.onnx` | Copied from the OpenCV Zoo release (`opencv/opencv_zoo`, `models/face_detection_yunet/`), the upstream artifact pinned by `manifests/default.json`. The only third-party weight committed to the repository; redistribution is permitted by the MIT license below. | MIT, Copyright (c) 2020 Shiqi Yu (the model directory's LICENSE file, reproduced below). |
| `face-portrait.jpg` | 500px rendition of a public-domain NASA portrait (Kalpana Chawla, official astronaut portrait), fetched from Wikimedia Commons (`File:Kalpana Chawla, NASA photo portrait in orange suit.jpg`), re-encoded as JPEG quality 75. | Public domain (work of NASA). Committed as the face-bearing image for the detection job tests (Task 6); the repository carries no other portrait fixture. |
| `face-portrait-orientation6.jpg` | Derived from `face-portrait.jpg` (Task 6): the upright frame rotated a quarter turn and re-encoded as JPEG quality 90 with EXIF orientation 6, so the stored sensor frame is 625x500 and a decoder honoring the tag restores the upright 500x625 frame. Generated once with libvips; the generation round-trip (reload, AutoRotate, compare) was verified before committing. | Public domain (work of NASA), as the derivative of the portrait above. Pins the detection job's EXIF-orientation handling: the face must land where the upright fixture's face lands. |

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

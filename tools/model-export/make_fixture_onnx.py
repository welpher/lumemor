#!/usr/bin/env python3
"""Generate the self-made ONNX fixtures used by the server ML tests.

The fixtures are hand-built minimal graphs -- zero third-party
weights are involved:

- identity.onnx: echoes a rank-1 float tensor ("input" -> "output").
  The session round-trip fixture: load it, run it on any float vector,
  get the same vector back.
- const128.onnx / const512.onnx / const768.onnx: map any NCHW float
  image tensor (1x3xHxW, dynamic H and W) to a constant vector of ones
  with the given dimension ("input" -> "output"). They stand in for the
  face recognition (dim 128) and CLIP embedding (dim 512) models in
  wiring tests; const768 additionally drives the heterogeneous-dimension
  model switch test (activating a 768-dim clip model must clear and
  re-dimension the embeddings column). The raw output is deliberately
  NOT unit-length (L2 norm sqrt(dim)), so the embedding wrappers'
  normalization is observable.
- ocr_det.onnx: the OCR detection fixture: a FIXED
  1x3x48x64 float input (the declared geometry the wrapper must honor)
  answered by a constant probability map -- four rectangular regions:
  three at 0.9 (two words of one row, one word of a second row) and one
  at 0.4 (a component the postprocess's score filter must drop). The
  binarize/score thresholds and the reading-order sort all have known
  expectations over this map.
- ocr_rec.onnx: the OCR recognition fixture: any NCHW
  float image tensor (1x3xHxW, dynamic H and W -- the recognition
  family's input convention) answered by a FIXED one-hot CTC sequence
  of shape [1, T, 6625]: the class indices per timestep are [blank,
  O, O, blank, C, blank, R, R, blank, blank], so the greedy decode
  exercises the blank-dropping and the repeat-collapsing rules and
  answers "OCR". 6625 matches the committed dictionary's class count
  (blank + 6623 dictionary lines + space); the class indices are the
  measured positions of O/C/R in that dictionary.

Outputs land in tools/models/cache/fixtures/ (git-ignored). They are
NOT committed by the generating task: a later task copies them into
server/internal/ml/testdata/ and commits them there.

Requires the Python `onnx` package (pip install onnx).
"""

import sys
from pathlib import Path

try:
    import onnx
    from onnx import TensorProto, helper
except ImportError:
    sys.exit("error: the onnx package is required to build fixtures (pip install onnx)")

# A conservative opset/IR pair: supported by every onnxruntime release
# the project can plausibly run (>= 1.12), and by current tooling.
OPSET = 17
IR_VERSION = 8

# tools/model-export/../models/cache/fixtures relative to this file.
OUTPUT_DIR = Path(__file__).resolve().parent.parent / "models" / "cache" / "fixtures"

# The OCR recognition fixture's geometry: the CTC sequence length and
# the class count. The class count equals the committed dictionary's
# table (blank + 6623 lines + space); the sequence carries a blank
# start, a repeated class, blank separators and a blank tail so the
# greedy decode's collapse rules are load-bearing.
OCR_REC_TIMESTEPS = 10
OCR_REC_CLASSES = 6625
OCR_REC_SEQUENCE = [0, 4741, 4741, 0, 4244, 0, 5127, 5127, 0, 0]
# (0 = blank; 4741/4244/5127 = the measured classes of O/C/R in the
# committed ppocr_keys_v1.txt dictionary.)


def identity_graph() -> onnx.ModelProto:
    """Build the echo fixture: output = input, any float length."""
    inp = helper.make_tensor_value_info("input", TensorProto.FLOAT, ["n"])
    out = helper.make_tensor_value_info("output", TensorProto.FLOAT, ["n"])
    node = helper.make_node("Identity", ["input"], ["output"])
    graph = helper.make_graph([node], "identity_fixture", [inp], [out])
    return finalize(graph)


def const_graph(dim: int) -> onnx.ModelProto:
    """Build a constant-output embedding fixture.

    The input image tensor is genuinely consumed (reduced to a scalar,
    then zeroed) so the graph signature stays image-in/vector-out, but
    the output is always `dim` ones regardless of the input -- the
    fixture stays deterministic for norm assertions.
    """
    inp = helper.make_tensor_value_info("input", TensorProto.FLOAT, [1, 3, "h", "w"])
    out = helper.make_tensor_value_info("output", TensorProto.FLOAT, [dim])
    nodes = [
        helper.make_node("ReduceSum", ["input"], ["sum"], keepdims=0),
        helper.make_node("Mul", ["sum", "zero"], ["zeroed"]),
        helper.make_node("Add", ["ones", "zeroed"], ["output"]),
    ]
    ones = helper.make_tensor("ones", TensorProto.FLOAT, [dim], [1.0] * dim)
    zero = helper.make_tensor("zero", TensorProto.FLOAT, [], [0.0])
    graph = helper.make_graph(
        nodes, f"const{dim}_fixture", [inp], [out], initializer=[ones, zero]
    )
    return finalize(graph)


def ocr_det_graph() -> onnx.ModelProto:
    """Build the OCR detection fixture.

    The input is a FIXED 1x3x48x64 float raster (the wrapper reads the
    declared geometry from the session metadata and resizes the image
    to it); the output is the constant [1, 1, 48, 64] probability map
    with the four regions documented in the module docstring. The input
    is consumed the same way the const fixtures consume theirs, so the
    map's value stays input-independent while the signature stays
    image-in/map-out.
    """
    height, width = 48, 64
    regions = [
        # (x1, y1, x2, y2, value) in map coordinates -- three kept
        # words (two of one row, one of the next) plus the score
        # filter's victim.
        (4, 8, 20, 16, 0.9),
        (30, 8, 50, 16, 0.9),
        (10, 40, 40, 48, 0.9),
        (52, 20, 62, 28, 0.4),
    ]
    values = [0.0] * (height * width)
    for x1, y1, x2, y2, value in regions:
        for y in range(y1, y2):
            for x in range(x1, x2):
                values[y * width + x] = value

    inp = helper.make_tensor_value_info("input", TensorProto.FLOAT, [1, 3, height, width])
    out = helper.make_tensor_value_info("output", TensorProto.FLOAT, [1, 1, height, width])
    nodes = [
        helper.make_node("ReduceSum", ["input"], ["sum"], keepdims=0),
        helper.make_node("Mul", ["sum", "zero"], ["zeroed"]),
        helper.make_node("Add", ["map", "zeroed"], ["output"]),
    ]
    prob = helper.make_tensor("map", TensorProto.FLOAT, [1, 1, height, width], values)
    zero = helper.make_tensor("zero", TensorProto.FLOAT, [], [0.0])
    graph = helper.make_graph(
        nodes, "ocr_det_fixture", [inp], [out], initializer=[prob, zero]
    )
    return finalize(graph)


def ocr_rec_graph() -> onnx.ModelProto:
    """Build the OCR recognition fixture.

    Any 1x3xHxW float raster in (dynamic H and W, the recognition
    family's input convention); out: a FIXED one-hot CTC sequence of
    shape [1, T, 6625] built by OneHot over the constant class-index
    sequence -- one-hot logits are exactly a softmax-style argmax
    candidate, so the greedy decode reads the sequence as written. The
    input is consumed (reduced, zeroed) and added onto the one-hot
    tensor as a broadcast scalar, keeping the signature honest without
    touching the values.
    """
    inp = helper.make_tensor_value_info("input", TensorProto.FLOAT, [1, 3, "h", "w"])
    out = helper.make_tensor_value_info(
        "output", TensorProto.FLOAT, [1, OCR_REC_TIMESTEPS, OCR_REC_CLASSES]
    )
    nodes = [
        helper.make_node("ReduceSum", ["input"], ["sum"], keepdims=0),
        helper.make_node("Mul", ["sum", "zero"], ["zeroed"]),
        helper.make_node("OneHot", ["indices", "depth", "values"], ["onehot"]),
        helper.make_node("Add", ["onehot", "zeroed"], ["output"]),
    ]
    indices = helper.make_tensor(
        "indices", TensorProto.INT64, [1, OCR_REC_TIMESTEPS], OCR_REC_SEQUENCE
    )
    depth = helper.make_tensor("depth", TensorProto.INT64, [], [OCR_REC_CLASSES])
    values = helper.make_tensor("values", TensorProto.FLOAT, [2], [0.0, 1.0])
    zero = helper.make_tensor("zero", TensorProto.FLOAT, [], [0.0])
    graph = helper.make_graph(
        nodes,
        "ocr_rec_fixture",
        [inp],
        [out],
        initializer=[indices, depth, values, zero],
    )
    return finalize(graph)


def text_embed_graph() -> onnx.GraphProto:
    """Build text_embed.onnx, the CLIP text-tower fixture.

    A stand-in for the Chinese-CLIP text tower with the real graph's
    input/output contract: three named int64 inputs
    (input_ids/attention_mask/token_type_ids) of shape [batch,
    tokens] -- the token extent dynamic, like the pinned tower's --
    answered by one [batch, 512] float32 feature named
    text_features. The feature is a DETERMINISTIC function of the
    token ids so the wiring tests stay sharp: feature d of row n is
    input_ids[n][d mod CONTEXT] + 1 (the gather wraps the context
    length; the +1 keeps every component nonzero so the wrapper's L2
    normalization is observable over a deliberately non-unit norm).

    The three inputs are consumed by name: attention_mask and
    token_type_ids feed a ReduceSum that the output adds as a zero
    broadcast (attention/token_type tensors of the wrapper's shape sum
    to a constant the arithmetic cancels), keeping the input contract
    honest without masking the token dependence.
    """
    CONTEXT = 52
    DIM = 512
    inp_ids = helper.make_tensor_value_info(
        "input_ids", TensorProto.INT64, ["n", "t"]
    )
    inp_mask = helper.make_tensor_value_info(
        "attention_mask", TensorProto.INT64, ["n", "t"]
    )
    inp_types = helper.make_tensor_value_info(
        "token_type_ids", TensorProto.INT64, ["n", "t"]
    )
    out = helper.make_tensor_value_info(
        "text_features", TensorProto.FLOAT, ["n", DIM]
    )
    indices = helper.make_tensor(
        "gather_indices", TensorProto.INT64, [DIM], [i % CONTEXT for i in range(DIM)]
    )
    one = helper.make_tensor("one", TensorProto.FLOAT, [], [1.0])
    zero = helper.make_tensor("zero", TensorProto.FLOAT, [], [0.0])
    nodes = [
        helper.make_node("Gather", ["input_ids", "gather_indices"], ["gathered"], axis=1),
        helper.make_node("Cast", ["gathered"], ["ids_f"], to=TensorProto.FLOAT),
        helper.make_node("Add", ["ids_f", "one"], ["plus_one"]),
        helper.make_node("ReduceSum", ["attention_mask"], ["mask_sum"], keepdims=1),
        helper.make_node("Cast", ["mask_sum"], ["mask_f"], to=TensorProto.FLOAT),
        helper.make_node("Mul", ["mask_f", "zero"], ["mask_zeroed"]),
        helper.make_node("ReduceSum", ["token_type_ids"], ["types_sum"], keepdims=1),
        helper.make_node("Cast", ["types_sum"], ["types_f"], to=TensorProto.FLOAT),
        helper.make_node("Mul", ["types_f", "zero"], ["types_zeroed"]),
        helper.make_node("Add", ["plus_one", "mask_zeroed"], ["with_mask"]),
        helper.make_node("Add", ["with_mask", "types_zeroed"], ["text_features"]),
    ]
    graph = helper.make_graph(
        nodes,
        "text_embed_fixture",
        [inp_ids, inp_mask, inp_types],
        [out],
        initializer=[indices, one, zero],
    )
    return finalize(graph)


def finalize(graph: onnx.GraphProto) -> onnx.ModelProto:
    """Stamp the shared opset/IR, validate, and return the model."""
    model = helper.make_model(
        graph,
        producer_name="make_fixture_onnx",
        opset_imports=[helper.make_opsetid("", OPSET)],
    )
    model.ir_version = IR_VERSION
    onnx.checker.check_model(model)
    return model


def main() -> None:
    fixtures = {
        "identity.onnx": identity_graph(),
        "const128.onnx": const_graph(128),
        "const512.onnx": const_graph(512),
        "const768.onnx": const_graph(768),
        "ocr_det.onnx": ocr_det_graph(),
        "ocr_rec.onnx": ocr_rec_graph(),
        "text_embed.onnx": text_embed_graph(),
    }
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    for name, model in fixtures.items():
        path = OUTPUT_DIR / name
        onnx.save(model, path)
        print(f"wrote {path} ({path.stat().st_size} bytes)")


if __name__ == "__main__":
    main()

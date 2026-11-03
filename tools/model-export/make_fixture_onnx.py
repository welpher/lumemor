#!/usr/bin/env python3
"""Generate the self-made ONNX fixtures used by the server ML tests.

The three fixtures are hand-built minimal graphs -- zero third-party
weights are involved:

- identity.onnx: echoes a rank-1 float tensor ("input" -> "output").
  The session round-trip fixture: load it, run it on any float vector,
  get the same vector back.
- const128.onnx / const512.onnx: map any NCHW float image tensor
  (1x3xHxW, dynamic H and W) to a constant vector of ones with the
  given dimension ("input" -> "output"). They stand in for the face
  recognition (dim 128) and CLIP embedding (dim 512) models in wiring
  tests; the raw output is deliberately NOT unit-length (L2 norm
  sqrt(dim)), so the embedding wrappers' normalization is observable.

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
    }
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    for name, model in fixtures.items():
        path = OUTPUT_DIR / name
        onnx.save(model, path)
        print(f"wrote {path} ({path.stat().st_size} bytes)")


if __name__ == "__main__":
    main()

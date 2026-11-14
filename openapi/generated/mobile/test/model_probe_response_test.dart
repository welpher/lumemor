import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';

// tests for ModelProbeResponse
void main() {
  final instance = ModelProbeResponseBuilder();
  // TODO add properties to the builder and call build()

  group(ModelProbeResponse, () {
    // String probeId
    test('to test the property `probeId`', () async {
      // TODO
    });

    // String state
    test('to test the property `state`', () async {
      // TODO
    });

    // ModelProbeResponseProgress progress
    test('to test the property `progress`', () async {
      // TODO
    });

    // The model's output dimension, read off the graph.
    // int dims
    test('to test the property `dims`', () async {
      // TODO
    });

    // ModelProbeResponseInputSize inputSize
    test('to test the property `inputSize`', () async {
      // TODO
    });

    // The staged bytes' digest, set when the probe finished (ok or warn).
    // String sha256
    test('to test the property `sha256`', () async {
      // TODO
    });

    // Whether the source's host is on the download sources list.
    // bool hostDeclared
    test('to test the property `hostDeclared`', () async {
      // TODO
    });

    // The failing check, named, for state `error`.
    // String reason
    test('to test the property `reason`', () async {
      // TODO
    });

  });
}

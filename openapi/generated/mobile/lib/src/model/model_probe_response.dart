//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/model_probe_response_progress.dart';
import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/model/model_probe_response_input_size.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_probe_response.g.dart';

/// One source probe's current state. `state` is `queued`, `running` (with the download progress when the source is a URL), `ok`, `error` (the reason names the check that failed) or `warn` (the checks passed; the host is not on the download sources list; the save may still proceed). `dims` and `inputSize` are the facts a finished probe read off the graph; `sha256` is the computed digest of the staged bytes (the given pin, when one was given, already matched it); a failed probe carries none of them. 
///
/// Properties:
/// * [probeId] 
/// * [state] 
/// * [progress] 
/// * [dims] - The model's output dimension, read off the graph.
/// * [inputSize] 
/// * [sha256] - The staged bytes' digest, set when the probe finished (ok or warn).
/// * [hostDeclared] - Whether the source's host is on the download sources list.
/// * [reason] - The failing check, named, for state `error`.
@BuiltValue()
abstract class ModelProbeResponse implements Built<ModelProbeResponse, ModelProbeResponseBuilder> {
  @BuiltValueField(wireName: r'probeId')
  String get probeId;

  @BuiltValueField(wireName: r'state')
  ModelProbeResponseStateEnum get state;
  // enum stateEnum {  queued,  running,  ok,  error,  warn,  };

  @BuiltValueField(wireName: r'progress')
  ModelProbeResponseProgress? get progress;

  /// The model's output dimension, read off the graph.
  @BuiltValueField(wireName: r'dims')
  int? get dims;

  @BuiltValueField(wireName: r'inputSize')
  ModelProbeResponseInputSize? get inputSize;

  /// The staged bytes' digest, set when the probe finished (ok or warn).
  @BuiltValueField(wireName: r'sha256')
  String? get sha256;

  /// Whether the source's host is on the download sources list.
  @BuiltValueField(wireName: r'hostDeclared')
  bool get hostDeclared;

  /// The failing check, named, for state `error`.
  @BuiltValueField(wireName: r'reason')
  String? get reason;

  ModelProbeResponse._();

  factory ModelProbeResponse([void updates(ModelProbeResponseBuilder b)]) = _$ModelProbeResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelProbeResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelProbeResponse> get serializer => _$ModelProbeResponseSerializer();
}

class _$ModelProbeResponseSerializer implements PrimitiveSerializer<ModelProbeResponse> {
  @override
  final Iterable<Type> types = const [ModelProbeResponse, _$ModelProbeResponse];

  @override
  final String wireName = r'ModelProbeResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelProbeResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'probeId';
    yield serializers.serialize(
      object.probeId,
      specifiedType: const FullType(String),
    );
    yield r'state';
    yield serializers.serialize(
      object.state,
      specifiedType: const FullType(ModelProbeResponseStateEnum),
    );
    yield r'progress';
    yield object.progress == null ? null : serializers.serialize(
      object.progress,
      specifiedType: const FullType.nullable(ModelProbeResponseProgress),
    );
    yield r'dims';
    yield object.dims == null ? null : serializers.serialize(
      object.dims,
      specifiedType: const FullType.nullable(int),
    );
    yield r'inputSize';
    yield object.inputSize == null ? null : serializers.serialize(
      object.inputSize,
      specifiedType: const FullType.nullable(ModelProbeResponseInputSize),
    );
    yield r'sha256';
    yield object.sha256 == null ? null : serializers.serialize(
      object.sha256,
      specifiedType: const FullType.nullable(String),
    );
    yield r'hostDeclared';
    yield serializers.serialize(
      object.hostDeclared,
      specifiedType: const FullType(bool),
    );
    yield r'reason';
    yield object.reason == null ? null : serializers.serialize(
      object.reason,
      specifiedType: const FullType.nullable(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ModelProbeResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelProbeResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'probeId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.probeId = valueDes;
          break;
        case r'state':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(ModelProbeResponseStateEnum),
          ) as ModelProbeResponseStateEnum;
          result.state = valueDes;
          break;
        case r'progress':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(ModelProbeResponseProgress),
          ) as ModelProbeResponseProgress?;
          if (valueDes == null) continue;
          result.progress.replace(valueDes);
          break;
        case r'dims':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(int),
          ) as int?;
          if (valueDes == null) continue;
          result.dims = valueDes;
          break;
        case r'inputSize':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(ModelProbeResponseInputSize),
          ) as ModelProbeResponseInputSize?;
          if (valueDes == null) continue;
          result.inputSize.replace(valueDes);
          break;
        case r'sha256':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.sha256 = valueDes;
          break;
        case r'hostDeclared':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.hostDeclared = valueDes;
          break;
        case r'reason':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.reason = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ModelProbeResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelProbeResponseBuilder();
    final serializedList = (serialized as Iterable<Object?>).toList();
    final unhandled = <Object?>[];
    _deserializeProperties(
      serializers,
      serialized,
      specifiedType: specifiedType,
      serializedList: serializedList,
      unhandled: unhandled,
      result: result,
    );
    return result.build();
  }
}


class ModelProbeResponseStateEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'queued')
  static const ModelProbeResponseStateEnum queued = _$modelProbeResponseStateEnum_queued;
  @BuiltValueEnumConst(wireName: r'running')
  static const ModelProbeResponseStateEnum running = _$modelProbeResponseStateEnum_running;
  @BuiltValueEnumConst(wireName: r'ok')
  static const ModelProbeResponseStateEnum ok = _$modelProbeResponseStateEnum_ok;
  @BuiltValueEnumConst(wireName: r'error')
  static const ModelProbeResponseStateEnum error = _$modelProbeResponseStateEnum_error;
  @BuiltValueEnumConst(wireName: r'warn')
  static const ModelProbeResponseStateEnum warn = _$modelProbeResponseStateEnum_warn;

  static Serializer<ModelProbeResponseStateEnum> get serializer => _$modelProbeResponseStateEnumSerializer;

  const ModelProbeResponseStateEnum._(String name): super(name);

  static BuiltSet<ModelProbeResponseStateEnum> get values => _$modelProbeResponseStateEnumValues;
  static ModelProbeResponseStateEnum valueOf(String name) => _$modelProbeResponseStateEnumValueOf(name);
}


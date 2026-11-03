//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_download_response.g.dart';

/// The identifier of one model download run: the `job_runs` row the download drives and the `job.*` events announce. 
///
/// Properties:
/// * [jobRunId] - UUIDv7 identifier of the run's `job_runs` row. `202` carries the freshly created run; `200` the existing in-flight or most recent one. 
@BuiltValue()
abstract class ModelDownloadResponse implements Built<ModelDownloadResponse, ModelDownloadResponseBuilder> {
  /// UUIDv7 identifier of the run's `job_runs` row. `202` carries the freshly created run; `200` the existing in-flight or most recent one. 
  @BuiltValueField(wireName: r'jobRunId')
  String get jobRunId;

  ModelDownloadResponse._();

  factory ModelDownloadResponse([void updates(ModelDownloadResponseBuilder b)]) = _$ModelDownloadResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelDownloadResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelDownloadResponse> get serializer => _$ModelDownloadResponseSerializer();
}

class _$ModelDownloadResponseSerializer implements PrimitiveSerializer<ModelDownloadResponse> {
  @override
  final Iterable<Type> types = const [ModelDownloadResponse, _$ModelDownloadResponse];

  @override
  final String wireName = r'ModelDownloadResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelDownloadResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'jobRunId';
    yield serializers.serialize(
      object.jobRunId,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ModelDownloadResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelDownloadResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'jobRunId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.jobRunId = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ModelDownloadResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelDownloadResponseBuilder();
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



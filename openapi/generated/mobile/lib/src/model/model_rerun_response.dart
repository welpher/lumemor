//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'model_rerun_response.g.dart';

/// The identifier of one rerun run: the `job_runs` row the rerun drives and the `job.*` events announce. 
///
/// Properties:
/// * [jobRunId] - UUIDv7 identifier of the run's `job_runs` row. `202` carries the freshly created run; `200` the in-flight one the click joined instead. 
@BuiltValue()
abstract class ModelRerunResponse implements Built<ModelRerunResponse, ModelRerunResponseBuilder> {
  /// UUIDv7 identifier of the run's `job_runs` row. `202` carries the freshly created run; `200` the in-flight one the click joined instead. 
  @BuiltValueField(wireName: r'jobRunId')
  String get jobRunId;

  ModelRerunResponse._();

  factory ModelRerunResponse([void updates(ModelRerunResponseBuilder b)]) = _$ModelRerunResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ModelRerunResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ModelRerunResponse> get serializer => _$ModelRerunResponseSerializer();
}

class _$ModelRerunResponseSerializer implements PrimitiveSerializer<ModelRerunResponse> {
  @override
  final Iterable<Type> types = const [ModelRerunResponse, _$ModelRerunResponse];

  @override
  final String wireName = r'ModelRerunResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ModelRerunResponse object, {
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
    ModelRerunResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ModelRerunResponseBuilder result,
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
  ModelRerunResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ModelRerunResponseBuilder();
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



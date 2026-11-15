//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'rebuild_derivatives_response.g.dart';

/// The identifier of one derivative rebuild run: the `job_runs` row the rebuild job drives and the `job.*` events announce. 
///
/// Properties:
/// * [jobRunId] - UUIDv7 identifier of the run's `job_runs` row. `202` carries the freshly created run; `200` the in-flight one the request joined instead, and the stop's `200` the run the stop landed on. 
@BuiltValue()
abstract class RebuildDerivativesResponse implements Built<RebuildDerivativesResponse, RebuildDerivativesResponseBuilder> {
  /// UUIDv7 identifier of the run's `job_runs` row. `202` carries the freshly created run; `200` the in-flight one the request joined instead, and the stop's `200` the run the stop landed on. 
  @BuiltValueField(wireName: r'jobRunId')
  String get jobRunId;

  RebuildDerivativesResponse._();

  factory RebuildDerivativesResponse([void updates(RebuildDerivativesResponseBuilder b)]) = _$RebuildDerivativesResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(RebuildDerivativesResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<RebuildDerivativesResponse> get serializer => _$RebuildDerivativesResponseSerializer();
}

class _$RebuildDerivativesResponseSerializer implements PrimitiveSerializer<RebuildDerivativesResponse> {
  @override
  final Iterable<Type> types = const [RebuildDerivativesResponse, _$RebuildDerivativesResponse];

  @override
  final String wireName = r'RebuildDerivativesResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    RebuildDerivativesResponse object, {
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
    RebuildDerivativesResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required RebuildDerivativesResponseBuilder result,
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
  RebuildDerivativesResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = RebuildDerivativesResponseBuilder();
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



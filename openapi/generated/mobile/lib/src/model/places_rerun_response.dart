//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'places_rerun_response.g.dart';

/// The identifier of one place re-resolution run: the `job_runs` row the re-resolution job drives and the `job.*` events announce. 
///
/// Properties:
/// * [jobRunId] - UUIDv7 identifier of the run's `job_runs` row. `202` carries the freshly created run; `200` the in-flight one the request joined instead. 
@BuiltValue()
abstract class PlacesRerunResponse implements Built<PlacesRerunResponse, PlacesRerunResponseBuilder> {
  /// UUIDv7 identifier of the run's `job_runs` row. `202` carries the freshly created run; `200` the in-flight one the request joined instead. 
  @BuiltValueField(wireName: r'jobRunId')
  String get jobRunId;

  PlacesRerunResponse._();

  factory PlacesRerunResponse([void updates(PlacesRerunResponseBuilder b)]) = _$PlacesRerunResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(PlacesRerunResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<PlacesRerunResponse> get serializer => _$PlacesRerunResponseSerializer();
}

class _$PlacesRerunResponseSerializer implements PrimitiveSerializer<PlacesRerunResponse> {
  @override
  final Iterable<Type> types = const [PlacesRerunResponse, _$PlacesRerunResponse];

  @override
  final String wireName = r'PlacesRerunResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    PlacesRerunResponse object, {
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
    PlacesRerunResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required PlacesRerunResponseBuilder result,
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
  PlacesRerunResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = PlacesRerunResponseBuilder();
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



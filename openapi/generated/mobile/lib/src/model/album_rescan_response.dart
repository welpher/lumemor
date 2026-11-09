//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_rescan_response.g.dart';

/// The identifier of one directory album rescan run: the `job_runs` row the rescan job drives and the `job.*` events announce. 
///
/// Properties:
/// * [jobRunId] - UUIDv7 identifier of the run's `job_runs` row. `202` carries the freshly created run; `200` the in-flight one of THIS album the request joined instead. 
@BuiltValue()
abstract class AlbumRescanResponse implements Built<AlbumRescanResponse, AlbumRescanResponseBuilder> {
  /// UUIDv7 identifier of the run's `job_runs` row. `202` carries the freshly created run; `200` the in-flight one of THIS album the request joined instead. 
  @BuiltValueField(wireName: r'jobRunId')
  String get jobRunId;

  AlbumRescanResponse._();

  factory AlbumRescanResponse([void updates(AlbumRescanResponseBuilder b)]) = _$AlbumRescanResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumRescanResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumRescanResponse> get serializer => _$AlbumRescanResponseSerializer();
}

class _$AlbumRescanResponseSerializer implements PrimitiveSerializer<AlbumRescanResponse> {
  @override
  final Iterable<Type> types = const [AlbumRescanResponse, _$AlbumRescanResponse];

  @override
  final String wireName = r'AlbumRescanResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumRescanResponse object, {
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
    AlbumRescanResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumRescanResponseBuilder result,
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
  AlbumRescanResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumRescanResponseBuilder();
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



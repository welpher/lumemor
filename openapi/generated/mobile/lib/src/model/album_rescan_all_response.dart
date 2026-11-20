//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_rescan_all_response.g.dart';

/// The identifier of one all-albums rescan run: the single `job_runs` row the sweep drives and the `job.*` events announce. The run's `total` is the instance's directory album count -- the history shows one row for the whole sweep, never one per album. 
///
/// Properties:
/// * [jobRunId] - UUIDv7 identifier of the run's `job_runs` row. `202` carries the freshly created sweep; `200` the in-flight one the request joined instead. 
@BuiltValue()
abstract class AlbumRescanAllResponse implements Built<AlbumRescanAllResponse, AlbumRescanAllResponseBuilder> {
  /// UUIDv7 identifier of the run's `job_runs` row. `202` carries the freshly created sweep; `200` the in-flight one the request joined instead. 
  @BuiltValueField(wireName: r'jobRunId')
  String get jobRunId;

  AlbumRescanAllResponse._();

  factory AlbumRescanAllResponse([void updates(AlbumRescanAllResponseBuilder b)]) = _$AlbumRescanAllResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumRescanAllResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumRescanAllResponse> get serializer => _$AlbumRescanAllResponseSerializer();
}

class _$AlbumRescanAllResponseSerializer implements PrimitiveSerializer<AlbumRescanAllResponse> {
  @override
  final Iterable<Type> types = const [AlbumRescanAllResponse, _$AlbumRescanAllResponse];

  @override
  final String wireName = r'AlbumRescanAllResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumRescanAllResponse object, {
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
    AlbumRescanAllResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumRescanAllResponseBuilder result,
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
  AlbumRescanAllResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumRescanAllResponseBuilder();
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



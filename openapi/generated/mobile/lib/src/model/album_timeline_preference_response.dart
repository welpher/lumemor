//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_timeline_preference_response.g.dart';

/// One album's timeline preference as the caller's row now carries it. Self-affecting state: the write publishes no event, this response is the only view that changed. 
///
/// Properties:
/// * [albumId] - Identifier of the album the preference scopes.
/// * [inTimeline] - Whether the album feeds the caller's aggregated timeline (a missing preference row means true, the default). 
@BuiltValue()
abstract class AlbumTimelinePreferenceResponse implements Built<AlbumTimelinePreferenceResponse, AlbumTimelinePreferenceResponseBuilder> {
  /// Identifier of the album the preference scopes.
  @BuiltValueField(wireName: r'albumId')
  String get albumId;

  /// Whether the album feeds the caller's aggregated timeline (a missing preference row means true, the default). 
  @BuiltValueField(wireName: r'inTimeline')
  bool get inTimeline;

  AlbumTimelinePreferenceResponse._();

  factory AlbumTimelinePreferenceResponse([void updates(AlbumTimelinePreferenceResponseBuilder b)]) = _$AlbumTimelinePreferenceResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumTimelinePreferenceResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumTimelinePreferenceResponse> get serializer => _$AlbumTimelinePreferenceResponseSerializer();
}

class _$AlbumTimelinePreferenceResponseSerializer implements PrimitiveSerializer<AlbumTimelinePreferenceResponse> {
  @override
  final Iterable<Type> types = const [AlbumTimelinePreferenceResponse, _$AlbumTimelinePreferenceResponse];

  @override
  final String wireName = r'AlbumTimelinePreferenceResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumTimelinePreferenceResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'albumId';
    yield serializers.serialize(
      object.albumId,
      specifiedType: const FullType(String),
    );
    yield r'inTimeline';
    yield serializers.serialize(
      object.inTimeline,
      specifiedType: const FullType(bool),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumTimelinePreferenceResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumTimelinePreferenceResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'albumId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.albumId = valueDes;
          break;
        case r'inTimeline':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.inTimeline = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AlbumTimelinePreferenceResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumTimelinePreferenceResponseBuilder();
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



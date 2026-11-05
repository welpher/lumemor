//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_timeline_preference_request.g.dart';

/// The caller's timeline preference write: the per-user switch of whether one album feeds the caller's aggregated timeline. The `inTimeline` field is required in practice: a body without it (or one carrying an explicit JSON null -- a required boolean has no absent-state semantics) answers 400 `VALIDATION_FAILED` naming the body. 
///
/// Properties:
/// * [inTimeline] - Whether the album's items feed the caller's aggregated timeline. `false` keeps them out; `true` (and a missing preference row) keeps them in. 
@BuiltValue()
abstract class AlbumTimelinePreferenceRequest implements Built<AlbumTimelinePreferenceRequest, AlbumTimelinePreferenceRequestBuilder> {
  /// Whether the album's items feed the caller's aggregated timeline. `false` keeps them out; `true` (and a missing preference row) keeps them in. 
  @BuiltValueField(wireName: r'inTimeline')
  bool? get inTimeline;

  AlbumTimelinePreferenceRequest._();

  factory AlbumTimelinePreferenceRequest([void updates(AlbumTimelinePreferenceRequestBuilder b)]) = _$AlbumTimelinePreferenceRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumTimelinePreferenceRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumTimelinePreferenceRequest> get serializer => _$AlbumTimelinePreferenceRequestSerializer();
}

class _$AlbumTimelinePreferenceRequestSerializer implements PrimitiveSerializer<AlbumTimelinePreferenceRequest> {
  @override
  final Iterable<Type> types = const [AlbumTimelinePreferenceRequest, _$AlbumTimelinePreferenceRequest];

  @override
  final String wireName = r'AlbumTimelinePreferenceRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumTimelinePreferenceRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    if (object.inTimeline != null) {
      yield r'inTimeline';
      yield serializers.serialize(
        object.inTimeline,
        specifiedType: const FullType(bool),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumTimelinePreferenceRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumTimelinePreferenceRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'inTimeline':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(bool),
          ) as bool?;
          if (valueDes == null) continue;
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
  AlbumTimelinePreferenceRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumTimelinePreferenceRequestBuilder();
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



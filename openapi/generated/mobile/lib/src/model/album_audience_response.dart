//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/model/album_share_entry.dart';
import 'package:lumemor_api/src/model/album_audience_skip.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_audience_response.g.dart';

/// The audience write's honest report: the audience as it stands after the write, the transitions this write caused, and the requested ids it skipped. The three id lists carry only what the write actually did -- a concurrent writer's rows surface in `audience`, never in a transition the response claims. 
///
/// Properties:
/// * [albumId] - Identifier of the album the audience belongs to.
/// * [audience] - The grantee list after the write, one entry per `album_shares` row in the grant order (`grantedAt` ascending) -- the same list the owner's album detail carries as `shares`, the edit dialog's chips. 
/// * [granted] - The ids whose rows THIS write landed -- empty for the idempotent repeat of the stored set. One `album.shared` event fired per entry. 
/// * [revoked] - The ids whose rows THIS write removed -- empty for the idempotent repeat. One `album.unshared` event fired per entry. 
/// * [skipped] - The requested ids that stayed ungranted, each with its reason. Empty when the whole request was actionable. 
@BuiltValue()
abstract class AlbumAudienceResponse implements Built<AlbumAudienceResponse, AlbumAudienceResponseBuilder> {
  /// Identifier of the album the audience belongs to.
  @BuiltValueField(wireName: r'albumId')
  String get albumId;

  /// The grantee list after the write, one entry per `album_shares` row in the grant order (`grantedAt` ascending) -- the same list the owner's album detail carries as `shares`, the edit dialog's chips. 
  @BuiltValueField(wireName: r'audience')
  BuiltList<AlbumShareEntry> get audience;

  /// The ids whose rows THIS write landed -- empty for the idempotent repeat of the stored set. One `album.shared` event fired per entry. 
  @BuiltValueField(wireName: r'granted')
  BuiltList<String> get granted;

  /// The ids whose rows THIS write removed -- empty for the idempotent repeat. One `album.unshared` event fired per entry. 
  @BuiltValueField(wireName: r'revoked')
  BuiltList<String> get revoked;

  /// The requested ids that stayed ungranted, each with its reason. Empty when the whole request was actionable. 
  @BuiltValueField(wireName: r'skipped')
  BuiltList<AlbumAudienceSkip> get skipped;

  AlbumAudienceResponse._();

  factory AlbumAudienceResponse([void updates(AlbumAudienceResponseBuilder b)]) = _$AlbumAudienceResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumAudienceResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumAudienceResponse> get serializer => _$AlbumAudienceResponseSerializer();
}

class _$AlbumAudienceResponseSerializer implements PrimitiveSerializer<AlbumAudienceResponse> {
  @override
  final Iterable<Type> types = const [AlbumAudienceResponse, _$AlbumAudienceResponse];

  @override
  final String wireName = r'AlbumAudienceResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumAudienceResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'albumId';
    yield serializers.serialize(
      object.albumId,
      specifiedType: const FullType(String),
    );
    yield r'audience';
    yield serializers.serialize(
      object.audience,
      specifiedType: const FullType(BuiltList, [FullType(AlbumShareEntry)]),
    );
    yield r'granted';
    yield serializers.serialize(
      object.granted,
      specifiedType: const FullType(BuiltList, [FullType(String)]),
    );
    yield r'revoked';
    yield serializers.serialize(
      object.revoked,
      specifiedType: const FullType(BuiltList, [FullType(String)]),
    );
    yield r'skipped';
    yield serializers.serialize(
      object.skipped,
      specifiedType: const FullType(BuiltList, [FullType(AlbumAudienceSkip)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumAudienceResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumAudienceResponseBuilder result,
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
        case r'audience':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(AlbumShareEntry)]),
          ) as BuiltList<AlbumShareEntry>;
          result.audience.replace(valueDes);
          break;
        case r'granted':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(String)]),
          ) as BuiltList<String>;
          result.granted.replace(valueDes);
          break;
        case r'revoked':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(String)]),
          ) as BuiltList<String>;
          result.revoked.replace(valueDes);
          break;
        case r'skipped':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(AlbumAudienceSkip)]),
          ) as BuiltList<AlbumAudienceSkip>;
          result.skipped.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AlbumAudienceResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumAudienceResponseBuilder();
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



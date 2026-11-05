//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_share_response.g.dart';

/// One share row as the grant endpoint answers it: the album, the grantee and the grant time. The idempotent re-grant answers the existing row's values -- the original `grantedAt` and grantor untouched. 
///
/// Properties:
/// * [albumId] - Identifier of the shared album.
/// * [userId] - Identifier of the grantee account.
/// * [grantedAt] - When the view ACL was granted (RFC 3339 UTC).
@BuiltValue()
abstract class AlbumShareResponse implements Built<AlbumShareResponse, AlbumShareResponseBuilder> {
  /// Identifier of the shared album.
  @BuiltValueField(wireName: r'albumId')
  String get albumId;

  /// Identifier of the grantee account.
  @BuiltValueField(wireName: r'userId')
  String get userId;

  /// When the view ACL was granted (RFC 3339 UTC).
  @BuiltValueField(wireName: r'grantedAt')
  DateTime get grantedAt;

  AlbumShareResponse._();

  factory AlbumShareResponse([void updates(AlbumShareResponseBuilder b)]) = _$AlbumShareResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumShareResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumShareResponse> get serializer => _$AlbumShareResponseSerializer();
}

class _$AlbumShareResponseSerializer implements PrimitiveSerializer<AlbumShareResponse> {
  @override
  final Iterable<Type> types = const [AlbumShareResponse, _$AlbumShareResponse];

  @override
  final String wireName = r'AlbumShareResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumShareResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'albumId';
    yield serializers.serialize(
      object.albumId,
      specifiedType: const FullType(String),
    );
    yield r'userId';
    yield serializers.serialize(
      object.userId,
      specifiedType: const FullType(String),
    );
    yield r'grantedAt';
    yield serializers.serialize(
      object.grantedAt,
      specifiedType: const FullType(DateTime),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumShareResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumShareResponseBuilder result,
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
        case r'userId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.userId = valueDes;
          break;
        case r'grantedAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DateTime),
          ) as DateTime;
          result.grantedAt = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AlbumShareResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumShareResponseBuilder();
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



//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_share_entry.g.dart';

/// One grantee of the owner's `shares` list on the album detail: the grantee's identity and the grant time -- the share management dialog's row. 
///
/// Properties:
/// * [userId] - Identifier of the grantee account.
/// * [email] - The grantee's login address.
/// * [grantedAt] - When the view ACL was granted (RFC 3339 UTC).
@BuiltValue()
abstract class AlbumShareEntry implements Built<AlbumShareEntry, AlbumShareEntryBuilder> {
  /// Identifier of the grantee account.
  @BuiltValueField(wireName: r'userId')
  String get userId;

  /// The grantee's login address.
  @BuiltValueField(wireName: r'email')
  String get email;

  /// When the view ACL was granted (RFC 3339 UTC).
  @BuiltValueField(wireName: r'grantedAt')
  DateTime get grantedAt;

  AlbumShareEntry._();

  factory AlbumShareEntry([void updates(AlbumShareEntryBuilder b)]) = _$AlbumShareEntry;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumShareEntryBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumShareEntry> get serializer => _$AlbumShareEntrySerializer();
}

class _$AlbumShareEntrySerializer implements PrimitiveSerializer<AlbumShareEntry> {
  @override
  final Iterable<Type> types = const [AlbumShareEntry, _$AlbumShareEntry];

  @override
  final String wireName = r'AlbumShareEntry';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumShareEntry object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'userId';
    yield serializers.serialize(
      object.userId,
      specifiedType: const FullType(String),
    );
    yield r'email';
    yield serializers.serialize(
      object.email,
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
    AlbumShareEntry object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumShareEntryBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'userId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.userId = valueDes;
          break;
        case r'email':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.email = valueDes;
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
  AlbumShareEntry deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumShareEntryBuilder();
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



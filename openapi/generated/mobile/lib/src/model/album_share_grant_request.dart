//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_share_grant_request.g.dart';

/// The grant body: the account receiving the view ACL. The target rules live in the endpoint's own answer -- the album's owner and unusable accounts answer 400 there, never a distinct shape here. 
///
/// Properties:
/// * [userId] - Identifier of the account receiving the view ACL.
@BuiltValue()
abstract class AlbumShareGrantRequest implements Built<AlbumShareGrantRequest, AlbumShareGrantRequestBuilder> {
  /// Identifier of the account receiving the view ACL.
  @BuiltValueField(wireName: r'userId')
  String get userId;

  AlbumShareGrantRequest._();

  factory AlbumShareGrantRequest([void updates(AlbumShareGrantRequestBuilder b)]) = _$AlbumShareGrantRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumShareGrantRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumShareGrantRequest> get serializer => _$AlbumShareGrantRequestSerializer();
}

class _$AlbumShareGrantRequestSerializer implements PrimitiveSerializer<AlbumShareGrantRequest> {
  @override
  final Iterable<Type> types = const [AlbumShareGrantRequest, _$AlbumShareGrantRequest];

  @override
  final String wireName = r'AlbumShareGrantRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumShareGrantRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'userId';
    yield serializers.serialize(
      object.userId,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumShareGrantRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumShareGrantRequestBuilder result,
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
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AlbumShareGrantRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumShareGrantRequestBuilder();
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



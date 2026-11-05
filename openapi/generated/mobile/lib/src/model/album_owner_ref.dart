//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_owner_ref.g.dart';

/// The owner of an album shared to the caller: the honest attribution a shared wall card renders. 
///
/// Properties:
/// * [id] - Identifier of the owning account.
/// * [email] - The email address of the owning account.
@BuiltValue()
abstract class AlbumOwnerRef implements Built<AlbumOwnerRef, AlbumOwnerRefBuilder> {
  /// Identifier of the owning account.
  @BuiltValueField(wireName: r'id')
  String get id;

  /// The email address of the owning account.
  @BuiltValueField(wireName: r'email')
  String get email;

  AlbumOwnerRef._();

  factory AlbumOwnerRef([void updates(AlbumOwnerRefBuilder b)]) = _$AlbumOwnerRef;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumOwnerRefBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumOwnerRef> get serializer => _$AlbumOwnerRefSerializer();
}

class _$AlbumOwnerRefSerializer implements PrimitiveSerializer<AlbumOwnerRef> {
  @override
  final Iterable<Type> types = const [AlbumOwnerRef, _$AlbumOwnerRef];

  @override
  final String wireName = r'AlbumOwnerRef';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumOwnerRef object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'email';
    yield serializers.serialize(
      object.email,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumOwnerRef object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumOwnerRefBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'id':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.id = valueDes;
          break;
        case r'email':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.email = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AlbumOwnerRef deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumOwnerRefBuilder();
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



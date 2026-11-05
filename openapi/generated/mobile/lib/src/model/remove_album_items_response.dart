//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'remove_album_items_response.g.dart';

/// The honest report of one membership remove over the request's distinct ids: the rows that left and the ids that were no members. 
///
/// Properties:
/// * [removed] - The membership rows the request deleted. Zero for an all-skipped request, which is still a 200. 
/// * [skipped] - The distinct ids that were no members of the album -- never added, or already removed. 
@BuiltValue()
abstract class RemoveAlbumItemsResponse implements Built<RemoveAlbumItemsResponse, RemoveAlbumItemsResponseBuilder> {
  /// The membership rows the request deleted. Zero for an all-skipped request, which is still a 200. 
  @BuiltValueField(wireName: r'removed')
  int get removed;

  /// The distinct ids that were no members of the album -- never added, or already removed. 
  @BuiltValueField(wireName: r'skipped')
  int get skipped;

  RemoveAlbumItemsResponse._();

  factory RemoveAlbumItemsResponse([void updates(RemoveAlbumItemsResponseBuilder b)]) = _$RemoveAlbumItemsResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(RemoveAlbumItemsResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<RemoveAlbumItemsResponse> get serializer => _$RemoveAlbumItemsResponseSerializer();
}

class _$RemoveAlbumItemsResponseSerializer implements PrimitiveSerializer<RemoveAlbumItemsResponse> {
  @override
  final Iterable<Type> types = const [RemoveAlbumItemsResponse, _$RemoveAlbumItemsResponse];

  @override
  final String wireName = r'RemoveAlbumItemsResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    RemoveAlbumItemsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'removed';
    yield serializers.serialize(
      object.removed,
      specifiedType: const FullType(int),
    );
    yield r'skipped';
    yield serializers.serialize(
      object.skipped,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    RemoveAlbumItemsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required RemoveAlbumItemsResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'removed':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.removed = valueDes;
          break;
        case r'skipped':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.skipped = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  RemoveAlbumItemsResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = RemoveAlbumItemsResponseBuilder();
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



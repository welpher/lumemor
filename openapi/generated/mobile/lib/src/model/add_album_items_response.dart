//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'add_album_items_response.g.dart';

/// The honest report of one membership add over the request's distinct ids: the rows that landed and the ids that stayed out. 
///
/// Properties:
/// * [added] - The membership rows the request landed -- the photos the creator currently sees and that were not members yet. Zero for an all-skipped request, which is still a 200. 
/// * [skipped] - The distinct ids that stayed out of the set: not currently visible to the caller (the re-share ground failed), not a live row, or already a member (the idempotent no-op). 
@BuiltValue()
abstract class AddAlbumItemsResponse implements Built<AddAlbumItemsResponse, AddAlbumItemsResponseBuilder> {
  /// The membership rows the request landed -- the photos the creator currently sees and that were not members yet. Zero for an all-skipped request, which is still a 200. 
  @BuiltValueField(wireName: r'added')
  int get added;

  /// The distinct ids that stayed out of the set: not currently visible to the caller (the re-share ground failed), not a live row, or already a member (the idempotent no-op). 
  @BuiltValueField(wireName: r'skipped')
  int get skipped;

  AddAlbumItemsResponse._();

  factory AddAlbumItemsResponse([void updates(AddAlbumItemsResponseBuilder b)]) = _$AddAlbumItemsResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AddAlbumItemsResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AddAlbumItemsResponse> get serializer => _$AddAlbumItemsResponseSerializer();
}

class _$AddAlbumItemsResponseSerializer implements PrimitiveSerializer<AddAlbumItemsResponse> {
  @override
  final Iterable<Type> types = const [AddAlbumItemsResponse, _$AddAlbumItemsResponse];

  @override
  final String wireName = r'AddAlbumItemsResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AddAlbumItemsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'added';
    yield serializers.serialize(
      object.added,
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
    AddAlbumItemsResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AddAlbumItemsResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'added':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.added = valueDes;
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
  AddAlbumItemsResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AddAlbumItemsResponseBuilder();
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



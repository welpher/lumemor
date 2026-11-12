//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_items_request.g.dart';

/// The membership write body, shared by the memory's curation arms: the list of asset ids to add to or remove from one memory album. Repeated ids collapse to one application; the report's counts cover the distinct ids. 
///
/// Properties:
/// * [assetIds] - The asset ids of the membership write, between 1 and 200 entries; the add skips the ids that are not currently visible to the caller or already members, the remove the ids that are no members. 
@BuiltValue()
abstract class AlbumItemsRequest implements Built<AlbumItemsRequest, AlbumItemsRequestBuilder> {
  /// The asset ids of the membership write, between 1 and 200 entries; the add skips the ids that are not currently visible to the caller or already members, the remove the ids that are no members. 
  @BuiltValueField(wireName: r'assetIds')
  BuiltList<String> get assetIds;

  AlbumItemsRequest._();

  factory AlbumItemsRequest([void updates(AlbumItemsRequestBuilder b)]) = _$AlbumItemsRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumItemsRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumItemsRequest> get serializer => _$AlbumItemsRequestSerializer();
}

class _$AlbumItemsRequestSerializer implements PrimitiveSerializer<AlbumItemsRequest> {
  @override
  final Iterable<Type> types = const [AlbumItemsRequest, _$AlbumItemsRequest];

  @override
  final String wireName = r'AlbumItemsRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumItemsRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'assetIds';
    yield serializers.serialize(
      object.assetIds,
      specifiedType: const FullType(BuiltList, [FullType(String)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumItemsRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumItemsRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'assetIds':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(String)]),
          ) as BuiltList<String>;
          result.assetIds.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AlbumItemsRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumItemsRequestBuilder();
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



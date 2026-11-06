//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_facet_count.g.dart';

/// One album of the albums facet.
///
/// Properties:
/// * [id] - The album id (the round-trip filter value).
/// * [title] - The album's title.
/// * [assetCount] - The number of members of the current filtered set (minus the album filter) this album holds. 
@BuiltValue()
abstract class AlbumFacetCount implements Built<AlbumFacetCount, AlbumFacetCountBuilder> {
  /// The album id (the round-trip filter value).
  @BuiltValueField(wireName: r'id')
  String get id;

  /// The album's title.
  @BuiltValueField(wireName: r'title')
  String get title;

  /// The number of members of the current filtered set (minus the album filter) this album holds. 
  @BuiltValueField(wireName: r'assetCount')
  int get assetCount;

  AlbumFacetCount._();

  factory AlbumFacetCount([void updates(AlbumFacetCountBuilder b)]) = _$AlbumFacetCount;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumFacetCountBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumFacetCount> get serializer => _$AlbumFacetCountSerializer();
}

class _$AlbumFacetCountSerializer implements PrimitiveSerializer<AlbumFacetCount> {
  @override
  final Iterable<Type> types = const [AlbumFacetCount, _$AlbumFacetCount];

  @override
  final String wireName = r'AlbumFacetCount';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumFacetCount object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'title';
    yield serializers.serialize(
      object.title,
      specifiedType: const FullType(String),
    );
    yield r'assetCount';
    yield serializers.serialize(
      object.assetCount,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumFacetCount object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumFacetCountBuilder result,
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
        case r'title':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.title = valueDes;
          break;
        case r'assetCount':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.assetCount = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AlbumFacetCount deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumFacetCountBuilder();
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



//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'tag_facet_count.g.dart';

/// One tag of the tags facet.
///
/// Properties:
/// * [id] - The CALLER's tag id (the round-trip filter value).
/// * [name] - The caller's tag name.
/// * [assetCount] - The number of assets of the current filtered set (minus the tag filter) carrying this tag. 
@BuiltValue()
abstract class TagFacetCount implements Built<TagFacetCount, TagFacetCountBuilder> {
  /// The CALLER's tag id (the round-trip filter value).
  @BuiltValueField(wireName: r'id')
  String get id;

  /// The caller's tag name.
  @BuiltValueField(wireName: r'name')
  String get name;

  /// The number of assets of the current filtered set (minus the tag filter) carrying this tag. 
  @BuiltValueField(wireName: r'assetCount')
  int get assetCount;

  TagFacetCount._();

  factory TagFacetCount([void updates(TagFacetCountBuilder b)]) = _$TagFacetCount;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TagFacetCountBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TagFacetCount> get serializer => _$TagFacetCountSerializer();
}

class _$TagFacetCountSerializer implements PrimitiveSerializer<TagFacetCount> {
  @override
  final Iterable<Type> types = const [TagFacetCount, _$TagFacetCount];

  @override
  final String wireName = r'TagFacetCount';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TagFacetCount object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'name';
    yield serializers.serialize(
      object.name,
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
    TagFacetCount object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TagFacetCountBuilder result,
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
        case r'name':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.name = valueDes;
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
  TagFacetCount deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TagFacetCountBuilder();
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



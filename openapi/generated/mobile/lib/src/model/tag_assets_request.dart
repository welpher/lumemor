//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'tag_assets_request.g.dart';

/// The assignment write body: the list of asset ids to tag with or untag from the caller's tag. Repeated ids collapse to one application; the report's counts cover the distinct ids. 
///
/// Properties:
/// * [assetIds] - The asset ids of the assignment write, between 1 and 200 entries. The add skips -- never errors on -- every id that is not caller-visible or that already carries the tag; the remove skips the ids that carry no assignment. 
@BuiltValue()
abstract class TagAssetsRequest implements Built<TagAssetsRequest, TagAssetsRequestBuilder> {
  /// The asset ids of the assignment write, between 1 and 200 entries. The add skips -- never errors on -- every id that is not caller-visible or that already carries the tag; the remove skips the ids that carry no assignment. 
  @BuiltValueField(wireName: r'assetIds')
  BuiltList<String> get assetIds;

  TagAssetsRequest._();

  factory TagAssetsRequest([void updates(TagAssetsRequestBuilder b)]) = _$TagAssetsRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TagAssetsRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TagAssetsRequest> get serializer => _$TagAssetsRequestSerializer();
}

class _$TagAssetsRequestSerializer implements PrimitiveSerializer<TagAssetsRequest> {
  @override
  final Iterable<Type> types = const [TagAssetsRequest, _$TagAssetsRequest];

  @override
  final String wireName = r'TagAssetsRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TagAssetsRequest object, {
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
    TagAssetsRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TagAssetsRequestBuilder result,
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
  TagAssetsRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TagAssetsRequestBuilder();
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



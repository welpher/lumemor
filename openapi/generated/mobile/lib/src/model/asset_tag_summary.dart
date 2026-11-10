//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'asset_tag_summary.g.dart';

/// One entry of the asset detail's `tags` array: the caller's own tag as it rides the asset -- identity and name only, the vocabulary's listing face carries the counts. Per-user state: the entry exists exactly because the CALLER's assignment row does. 
///
/// Properties:
/// * [id] - Identifier of the tag (the caller's own row).
/// * [name] - The tag's name, the caller's own vocabulary.
@BuiltValue()
abstract class AssetTagSummary implements Built<AssetTagSummary, AssetTagSummaryBuilder> {
  /// Identifier of the tag (the caller's own row).
  @BuiltValueField(wireName: r'id')
  String get id;

  /// The tag's name, the caller's own vocabulary.
  @BuiltValueField(wireName: r'name')
  String get name;

  AssetTagSummary._();

  factory AssetTagSummary([void updates(AssetTagSummaryBuilder b)]) = _$AssetTagSummary;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AssetTagSummaryBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AssetTagSummary> get serializer => _$AssetTagSummarySerializer();
}

class _$AssetTagSummarySerializer implements PrimitiveSerializer<AssetTagSummary> {
  @override
  final Iterable<Type> types = const [AssetTagSummary, _$AssetTagSummary];

  @override
  final String wireName = r'AssetTagSummary';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AssetTagSummary object, {
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
  }

  @override
  Object serialize(
    Serializers serializers,
    AssetTagSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AssetTagSummaryBuilder result,
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
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AssetTagSummary deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AssetTagSummaryBuilder();
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



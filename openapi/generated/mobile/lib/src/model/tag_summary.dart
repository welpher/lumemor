//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'tag_summary.g.dart';

/// One tag of the caller's vocabulary as every tags-face response answers it: the identity, the name, the honest count and the timestamps. The write faces (create, rename) answer this same shape, so a write's answer is the entry the listing lists. 
///
/// Properties:
/// * [id] - UUIDv7 identifier of the tag (the caller's own row).
/// * [name] - The tag's display name, unique within the caller's vocabulary. Trimmed on write; 1-64 Unicode characters (counted over runes). 
/// * [assetCount] - The tag's assignment rows joined to caller-visible live assets: the album union plus the timeline's live filters (not trashed, not archived, no Live Photo component). A trashed or archived member stops counting until it returns; the assignment row survives underneath, exactly like a favorite row. 
/// * [createdAt] - When the tag row was created (RFC 3339 UTC).
/// * [updatedAt] - When the tag's name last truly changed (RFC 3339 UTC). The idempotent create and the no-op rename leave it untouched -- only a real rename moves it. 
@BuiltValue()
abstract class TagSummary implements Built<TagSummary, TagSummaryBuilder> {
  /// UUIDv7 identifier of the tag (the caller's own row).
  @BuiltValueField(wireName: r'id')
  String get id;

  /// The tag's display name, unique within the caller's vocabulary. Trimmed on write; 1-64 Unicode characters (counted over runes). 
  @BuiltValueField(wireName: r'name')
  String get name;

  /// The tag's assignment rows joined to caller-visible live assets: the album union plus the timeline's live filters (not trashed, not archived, no Live Photo component). A trashed or archived member stops counting until it returns; the assignment row survives underneath, exactly like a favorite row. 
  @BuiltValueField(wireName: r'assetCount')
  int get assetCount;

  /// When the tag row was created (RFC 3339 UTC).
  @BuiltValueField(wireName: r'createdAt')
  DateTime get createdAt;

  /// When the tag's name last truly changed (RFC 3339 UTC). The idempotent create and the no-op rename leave it untouched -- only a real rename moves it. 
  @BuiltValueField(wireName: r'updatedAt')
  DateTime get updatedAt;

  TagSummary._();

  factory TagSummary([void updates(TagSummaryBuilder b)]) = _$TagSummary;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TagSummaryBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TagSummary> get serializer => _$TagSummarySerializer();
}

class _$TagSummarySerializer implements PrimitiveSerializer<TagSummary> {
  @override
  final Iterable<Type> types = const [TagSummary, _$TagSummary];

  @override
  final String wireName = r'TagSummary';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TagSummary object, {
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
    yield r'createdAt';
    yield serializers.serialize(
      object.createdAt,
      specifiedType: const FullType(DateTime),
    );
    yield r'updatedAt';
    yield serializers.serialize(
      object.updatedAt,
      specifiedType: const FullType(DateTime),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    TagSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TagSummaryBuilder result,
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
        case r'createdAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DateTime),
          ) as DateTime;
          result.createdAt = valueDes;
          break;
        case r'updatedAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DateTime),
          ) as DateTime;
          result.updatedAt = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  TagSummary deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TagSummaryBuilder();
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



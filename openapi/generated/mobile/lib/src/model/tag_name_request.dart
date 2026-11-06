//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'tag_name_request.g.dart';

/// The name-carrying body of the tag create and rename writes: one field, required. The name is trimmed server-side and must span 1-64 Unicode characters after the trim (the count is over runes, so multi-byte names stay within the bound at the same length). 
///
/// Properties:
/// * [name] - The tag's name, trimmed server-side. On create a name the caller already owns converges on the existing tag; on rename a name another tag carries answers 409 `TAG_NAME_TAKEN`, the row's own name the idempotent no-op. 
@BuiltValue()
abstract class TagNameRequest implements Built<TagNameRequest, TagNameRequestBuilder> {
  /// The tag's name, trimmed server-side. On create a name the caller already owns converges on the existing tag; on rename a name another tag carries answers 409 `TAG_NAME_TAKEN`, the row's own name the idempotent no-op. 
  @BuiltValueField(wireName: r'name')
  String get name;

  TagNameRequest._();

  factory TagNameRequest([void updates(TagNameRequestBuilder b)]) = _$TagNameRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TagNameRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TagNameRequest> get serializer => _$TagNameRequestSerializer();
}

class _$TagNameRequestSerializer implements PrimitiveSerializer<TagNameRequest> {
  @override
  final Iterable<Type> types = const [TagNameRequest, _$TagNameRequest];

  @override
  final String wireName = r'TagNameRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TagNameRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'name';
    yield serializers.serialize(
      object.name,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    TagNameRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TagNameRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
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
  TagNameRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TagNameRequestBuilder();
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



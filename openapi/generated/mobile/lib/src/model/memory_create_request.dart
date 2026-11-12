//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'memory_create_request.g.dart';

/// The creation body of one memory: the title and the audience the memory starts with, nothing else. A memory owns no source path, no deletability switch and no timeline switch -- the fields the directory album's creation carries have no home on a reference set, and the shape says so by omitting them. 
///
/// Properties:
/// * [title] - The memory's display title, trimmed server-side; it must span 1-64 Unicode characters after the trim (the count is over runes, so multi-byte titles stay within the bound at the same length). 
/// * [audience] - The accounts the memory starts being visible to: the initial `album_shares` set, empty (or absent) meaning \"only me\". Entries that cannot be granted -- an unknown or disabled account, or the caller's own id -- are skipped and reported in the response's `skipped` list, never errors. 
@BuiltValue()
abstract class MemoryCreateRequest implements Built<MemoryCreateRequest, MemoryCreateRequestBuilder> {
  /// The memory's display title, trimmed server-side; it must span 1-64 Unicode characters after the trim (the count is over runes, so multi-byte titles stay within the bound at the same length). 
  @BuiltValueField(wireName: r'title')
  String get title;

  /// The accounts the memory starts being visible to: the initial `album_shares` set, empty (or absent) meaning \"only me\". Entries that cannot be granted -- an unknown or disabled account, or the caller's own id -- are skipped and reported in the response's `skipped` list, never errors. 
  @BuiltValueField(wireName: r'audience')
  BuiltList<String>? get audience;

  MemoryCreateRequest._();

  factory MemoryCreateRequest([void updates(MemoryCreateRequestBuilder b)]) = _$MemoryCreateRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(MemoryCreateRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<MemoryCreateRequest> get serializer => _$MemoryCreateRequestSerializer();
}

class _$MemoryCreateRequestSerializer implements PrimitiveSerializer<MemoryCreateRequest> {
  @override
  final Iterable<Type> types = const [MemoryCreateRequest, _$MemoryCreateRequest];

  @override
  final String wireName = r'MemoryCreateRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    MemoryCreateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'title';
    yield serializers.serialize(
      object.title,
      specifiedType: const FullType(String),
    );
    if (object.audience != null) {
      yield r'audience';
      yield serializers.serialize(
        object.audience,
        specifiedType: const FullType(BuiltList, [FullType(String)]),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    MemoryCreateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required MemoryCreateRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'title':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.title = valueDes;
          break;
        case r'audience':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(BuiltList, [FullType(String)]),
          ) as BuiltList<String>?;
          if (valueDes == null) continue;
          result.audience.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  MemoryCreateRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = MemoryCreateRequestBuilder();
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



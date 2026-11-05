//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_create_request.g.dart';

/// The creation body of one virtual album: the title and the optional bounded description. No type field exists -- this endpoint IS virtual creation, the only creatable album type of the face. 
///
/// Properties:
/// * [title] - The album's display title, trimmed server-side; it must span 1-64 Unicode characters after the trim (the count is over runes, so multi-byte titles stay within the bound at the same length). 
/// * [description] - The album's free-text description; empty when absent. At most 10,000 Unicode characters, the asset description's own bound. 
@BuiltValue()
abstract class AlbumCreateRequest implements Built<AlbumCreateRequest, AlbumCreateRequestBuilder> {
  /// The album's display title, trimmed server-side; it must span 1-64 Unicode characters after the trim (the count is over runes, so multi-byte titles stay within the bound at the same length). 
  @BuiltValueField(wireName: r'title')
  String get title;

  /// The album's free-text description; empty when absent. At most 10,000 Unicode characters, the asset description's own bound. 
  @BuiltValueField(wireName: r'description')
  String? get description;

  AlbumCreateRequest._();

  factory AlbumCreateRequest([void updates(AlbumCreateRequestBuilder b)]) = _$AlbumCreateRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumCreateRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumCreateRequest> get serializer => _$AlbumCreateRequestSerializer();
}

class _$AlbumCreateRequestSerializer implements PrimitiveSerializer<AlbumCreateRequest> {
  @override
  final Iterable<Type> types = const [AlbumCreateRequest, _$AlbumCreateRequest];

  @override
  final String wireName = r'AlbumCreateRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumCreateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'title';
    yield serializers.serialize(
      object.title,
      specifiedType: const FullType(String),
    );
    if (object.description != null) {
      yield r'description';
      yield serializers.serialize(
        object.description,
        specifiedType: const FullType(String),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumCreateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumCreateRequestBuilder result,
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
        case r'description':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.description = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AlbumCreateRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumCreateRequestBuilder();
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



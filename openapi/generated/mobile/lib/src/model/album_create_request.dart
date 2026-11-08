//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_create_request.g.dart';

/// The creation body of one directory album (E4's albums-as-sources): the title, the existing directory the album references in place, and the audience the album starts with. No type field exists -- this endpoint IS directory creation, the only creatable album type; no description field either -- the dialog carries title, path and audience, and the description stays empty (the property patch can set one later). 
///
/// Properties:
/// * [title] - The album's display title, trimmed server-side; it must span 1-64 Unicode characters after the trim (the count is over runes, so multi-byte titles stay within the bound at the same length). 
/// * [rootPath] - The source directory: an existing absolute path the server validates against the path matrix (existence, directory-ness, readability, symlink resolution, the managed tree, root overlap, the mount-root allowlist) before creating anything. The album row stores the RESOLVED absolute path -- symlink segments evaluated -- so a later symlink swap cannot repoint the album. 
/// * [audience] - The accounts the album starts being visible to: the initial `album_shares` set, empty (or absent) meaning \"only me\". Entries that cannot be granted -- an unknown or disabled account, or the caller's own id -- are skipped and reported in the response's `skipped` list, never errors. 
@BuiltValue()
abstract class AlbumCreateRequest implements Built<AlbumCreateRequest, AlbumCreateRequestBuilder> {
  /// The album's display title, trimmed server-side; it must span 1-64 Unicode characters after the trim (the count is over runes, so multi-byte titles stay within the bound at the same length). 
  @BuiltValueField(wireName: r'title')
  String get title;

  /// The source directory: an existing absolute path the server validates against the path matrix (existence, directory-ness, readability, symlink resolution, the managed tree, root overlap, the mount-root allowlist) before creating anything. The album row stores the RESOLVED absolute path -- symlink segments evaluated -- so a later symlink swap cannot repoint the album. 
  @BuiltValueField(wireName: r'rootPath')
  String get rootPath;

  /// The accounts the album starts being visible to: the initial `album_shares` set, empty (or absent) meaning \"only me\". Entries that cannot be granted -- an unknown or disabled account, or the caller's own id -- are skipped and reported in the response's `skipped` list, never errors. 
  @BuiltValueField(wireName: r'audience')
  BuiltList<String>? get audience;

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
    yield r'rootPath';
    yield serializers.serialize(
      object.rootPath,
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
        case r'rootPath':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.rootPath = valueDes;
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



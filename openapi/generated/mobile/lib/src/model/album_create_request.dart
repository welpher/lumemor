//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_create_request.g.dart';

/// The creation body of one directory album (albums-as-sources): the title, the existing directory the album references in place, the audience the album starts with, and the two dialog switches of the create-time autonomy -- the deletability switch and the creator's timeline exclusion. No type field exists -- this endpoint IS directory creation, the only creatable album type. No description field either: it died whole (the 2026-11-08 owner evaluation -- real-world fill rate near zero, the context already carried by title/date-range+count/place/cover), and the property patch carries no field that could set one. 
///
/// Properties:
/// * [title] - The album's display title, trimmed server-side; it must span 1-64 Unicode characters after the trim (the count is over runes, so multi-byte titles stay within the bound at the same length). 
/// * [rootPath] - The source directory: an existing absolute path the server validates against the path matrix (existence, directory-ness, readability, symlink resolution, the managed tree) before creating anything. The album row stores the RESOLVED absolute path -- symlink segments evaluated -- so a later symlink swap cannot repoint the album. After the matrix passes, the server probes the directory's writability once and records the fact; a read-only directory is a valid source, never a refusal. 
/// * [allowAppWrites] - The owner's write switch, carried from the creation dialog: whether the application may modify the album's source files -- delete them through the trash move, and write a capture-time edit back into their metadata. On by default; the dialog renders the switch only where the source is writable (the completion row's `writable` fact), and on a read-only source the stored value governs nothing -- the derived `deletable` and `metadataWritable` faces stay false whatever it says. Editable later through the property patch. 
/// * [inTimeline] - The creator's timeline preference, carried from the creation dialog: `false` keeps the album out of the CREATOR's aggregated timeline from birth -- the same per-user switch `PUT /albums/{albumId}/timeline-preference` writes, landed in the creation itself (a per-user preference, not an album fact: it needs no photos to exist, so the creation is exactly the right moment). Absent or true feeds the timeline (the default -- no preference row is written). The exclusion is the creator's own: the audience members' feeds are governed by their own preference rows, never by this field. 
/// * [audience] - The accounts the album starts being visible to: the initial `album_shares` set, empty (or absent) meaning \"only me\". Entries that cannot be granted -- an unknown or disabled account, or the caller's own id -- are skipped and reported in the response's `skipped` list, never errors. 
@BuiltValue()
abstract class AlbumCreateRequest implements Built<AlbumCreateRequest, AlbumCreateRequestBuilder> {
  /// The album's display title, trimmed server-side; it must span 1-64 Unicode characters after the trim (the count is over runes, so multi-byte titles stay within the bound at the same length). 
  @BuiltValueField(wireName: r'title')
  String get title;

  /// The source directory: an existing absolute path the server validates against the path matrix (existence, directory-ness, readability, symlink resolution, the managed tree) before creating anything. The album row stores the RESOLVED absolute path -- symlink segments evaluated -- so a later symlink swap cannot repoint the album. After the matrix passes, the server probes the directory's writability once and records the fact; a read-only directory is a valid source, never a refusal. 
  @BuiltValueField(wireName: r'rootPath')
  String get rootPath;

  /// The owner's write switch, carried from the creation dialog: whether the application may modify the album's source files -- delete them through the trash move, and write a capture-time edit back into their metadata. On by default; the dialog renders the switch only where the source is writable (the completion row's `writable` fact), and on a read-only source the stored value governs nothing -- the derived `deletable` and `metadataWritable` faces stay false whatever it says. Editable later through the property patch. 
  @BuiltValueField(wireName: r'allowAppWrites')
  bool? get allowAppWrites;

  /// The creator's timeline preference, carried from the creation dialog: `false` keeps the album out of the CREATOR's aggregated timeline from birth -- the same per-user switch `PUT /albums/{albumId}/timeline-preference` writes, landed in the creation itself (a per-user preference, not an album fact: it needs no photos to exist, so the creation is exactly the right moment). Absent or true feeds the timeline (the default -- no preference row is written). The exclusion is the creator's own: the audience members' feeds are governed by their own preference rows, never by this field. 
  @BuiltValueField(wireName: r'inTimeline')
  bool? get inTimeline;

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
    if (object.allowAppWrites != null) {
      yield r'allowAppWrites';
      yield serializers.serialize(
        object.allowAppWrites,
        specifiedType: const FullType(bool),
      );
    }
    if (object.inTimeline != null) {
      yield r'inTimeline';
      yield serializers.serialize(
        object.inTimeline,
        specifiedType: const FullType(bool),
      );
    }
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
        case r'allowAppWrites':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(bool),
          ) as bool?;
          if (valueDes == null) continue;
          result.allowAppWrites = valueDes;
          break;
        case r'inTimeline':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(bool),
          ) as bool?;
          if (valueDes == null) continue;
          result.inTimeline = valueDes;
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



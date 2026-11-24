//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_patch.g.dart';

/// The partial edit of one album: every field is optional, at least one must be present (a body that carries none -- JSON null values count as absent, the patch discipline of the asset surface -- is a client bug and answers 400), and only the fields the request carries update; the rest keep their values. A present `coverAssetIds` must carry 0 or exactly the card's frame count of distinct ids, every one a renderable member of the album itself (an archived member is renderable on an album and refused on a memory). The write publishes one `album.updated` event whatever fields changed. 
///
/// Properties:
/// * [title] - The album's display title, trimmed server-side; when present it must span 1-64 Unicode characters after the trim. 
/// * [coverAssetIds] - The owner's pin, as a full set: the frames of the cover in render order (the order the owner picked them in -- the server stores and returns it as given, never re-sorting). The length is 0 or exactly N, N being the card's frame count (four on the album card, six on the memory band): an empty array clears the pin and returns the album to the automatic weekly group, a full set pins it, and any other length answers 400 `VALIDATION_FAILED` (a partial set is the picker's un-saveable state, never a stored one). Repeated ids answer 400 too -- a cover is a set, and one frame cannot fill two cells. Every id must be a RENDERABLE member of THIS album (the membership row exists and the asset is not trashed, is not a Live Photo component row, and on a memory album is not archived -- an album's archived members render and may pin); otherwise the answer is 400 with the message key `album_cover_not_member` and a `count` of the refused frames. Absent (or JSON null, which decodes as absent) leaves the pin unchanged. 
/// * [allowAppWrites] - The owner's write switch of a DIRECTORY album: whether the application may modify the album's source files -- delete them through the trash move, and write a capture-time edit back into their metadata. On for a writable source by default; the owner who manages the files themselves turns it off, and the wall's delete affordances and the assets' time edits disappear with it (the derived `deletable` and `metadataWritable` faces). A patch carrying the field on a memory album answers 400 with the message key `album_deletable_not_directory` -- the switch is the directory album's own, a memory album's deletions never move files. Idempotent: repeating the stored value converges. 
@BuiltValue()
abstract class AlbumPatch implements Built<AlbumPatch, AlbumPatchBuilder> {
  /// The album's display title, trimmed server-side; when present it must span 1-64 Unicode characters after the trim. 
  @BuiltValueField(wireName: r'title')
  String? get title;

  /// The owner's pin, as a full set: the frames of the cover in render order (the order the owner picked them in -- the server stores and returns it as given, never re-sorting). The length is 0 or exactly N, N being the card's frame count (four on the album card, six on the memory band): an empty array clears the pin and returns the album to the automatic weekly group, a full set pins it, and any other length answers 400 `VALIDATION_FAILED` (a partial set is the picker's un-saveable state, never a stored one). Repeated ids answer 400 too -- a cover is a set, and one frame cannot fill two cells. Every id must be a RENDERABLE member of THIS album (the membership row exists and the asset is not trashed, is not a Live Photo component row, and on a memory album is not archived -- an album's archived members render and may pin); otherwise the answer is 400 with the message key `album_cover_not_member` and a `count` of the refused frames. Absent (or JSON null, which decodes as absent) leaves the pin unchanged. 
  @BuiltValueField(wireName: r'coverAssetIds')
  BuiltList<String>? get coverAssetIds;

  /// The owner's write switch of a DIRECTORY album: whether the application may modify the album's source files -- delete them through the trash move, and write a capture-time edit back into their metadata. On for a writable source by default; the owner who manages the files themselves turns it off, and the wall's delete affordances and the assets' time edits disappear with it (the derived `deletable` and `metadataWritable` faces). A patch carrying the field on a memory album answers 400 with the message key `album_deletable_not_directory` -- the switch is the directory album's own, a memory album's deletions never move files. Idempotent: repeating the stored value converges. 
  @BuiltValueField(wireName: r'allowAppWrites')
  bool? get allowAppWrites;

  AlbumPatch._();

  factory AlbumPatch([void updates(AlbumPatchBuilder b)]) = _$AlbumPatch;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AlbumPatchBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AlbumPatch> get serializer => _$AlbumPatchSerializer();
}

class _$AlbumPatchSerializer implements PrimitiveSerializer<AlbumPatch> {
  @override
  final Iterable<Type> types = const [AlbumPatch, _$AlbumPatch];

  @override
  final String wireName = r'AlbumPatch';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AlbumPatch object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    if (object.title != null) {
      yield r'title';
      yield serializers.serialize(
        object.title,
        specifiedType: const FullType(String),
      );
    }
    if (object.coverAssetIds != null) {
      yield r'coverAssetIds';
      yield serializers.serialize(
        object.coverAssetIds,
        specifiedType: const FullType(BuiltList, [FullType(String)]),
      );
    }
    if (object.allowAppWrites != null) {
      yield r'allowAppWrites';
      yield serializers.serialize(
        object.allowAppWrites,
        specifiedType: const FullType(bool),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    AlbumPatch object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AlbumPatchBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'title':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.title = valueDes;
          break;
        case r'coverAssetIds':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(BuiltList, [FullType(String)]),
          ) as BuiltList<String>?;
          if (valueDes == null) continue;
          result.coverAssetIds.replace(valueDes);
          break;
        case r'allowAppWrites':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(bool),
          ) as bool?;
          if (valueDes == null) continue;
          result.allowAppWrites = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AlbumPatch deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AlbumPatchBuilder();
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



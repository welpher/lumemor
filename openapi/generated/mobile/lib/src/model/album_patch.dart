//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'album_patch.g.dart';

/// The partial edit of one album: every field is optional, at least one must be present (a body that carries none -- JSON null values count as absent, the patch discipline of the asset surface -- is a client bug and answers 400), and only the fields the request carries update; the rest keep their values. A present `coverAssetId` must be a live member of the album itself. The write publishes one `album.updated` event whatever fields changed. 
///
/// Properties:
/// * [title] - The album's display title, trimmed server-side; when present it must span 1-64 Unicode characters after the trim. 
/// * [coverAssetId] - The pinned cover asset. When present it must be a LIVE member of THIS album (the membership row exists and the asset is not trashed, not hidden, not a Live Photo component row) -- otherwise the answer is 400 with the message key `album_cover_not_member`. Absent (or JSON null, which decodes as absent) leaves the pin unchanged. 
/// * [allowAppDeletion] - The owner's deletability switch of a DIRECTORY album: whether the album's photos may be deleted through the application (the trash move of the source file). On for a writable source by default; the owner who manages file-system deletions themselves turns it off, and the wall's delete affordances disappear with it (the derived `deletable` face of the detail read). A patch carrying the field on a memory album answers 400 with the message key `album_deletable_not_directory` -- the switch is the directory album's own, a memory album's deletions never move files. Idempotent: repeating the stored value converges. 
@BuiltValue()
abstract class AlbumPatch implements Built<AlbumPatch, AlbumPatchBuilder> {
  /// The album's display title, trimmed server-side; when present it must span 1-64 Unicode characters after the trim. 
  @BuiltValueField(wireName: r'title')
  String? get title;

  /// The pinned cover asset. When present it must be a LIVE member of THIS album (the membership row exists and the asset is not trashed, not hidden, not a Live Photo component row) -- otherwise the answer is 400 with the message key `album_cover_not_member`. Absent (or JSON null, which decodes as absent) leaves the pin unchanged. 
  @BuiltValueField(wireName: r'coverAssetId')
  String? get coverAssetId;

  /// The owner's deletability switch of a DIRECTORY album: whether the album's photos may be deleted through the application (the trash move of the source file). On for a writable source by default; the owner who manages file-system deletions themselves turns it off, and the wall's delete affordances disappear with it (the derived `deletable` face of the detail read). A patch carrying the field on a memory album answers 400 with the message key `album_deletable_not_directory` -- the switch is the directory album's own, a memory album's deletions never move files. Idempotent: repeating the stored value converges. 
  @BuiltValueField(wireName: r'allowAppDeletion')
  bool? get allowAppDeletion;

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
    if (object.coverAssetId != null) {
      yield r'coverAssetId';
      yield serializers.serialize(
        object.coverAssetId,
        specifiedType: const FullType(String),
      );
    }
    if (object.allowAppDeletion != null) {
      yield r'allowAppDeletion';
      yield serializers.serialize(
        object.allowAppDeletion,
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
        case r'coverAssetId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.coverAssetId = valueDes;
          break;
        case r'allowAppDeletion':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(bool),
          ) as bool?;
          if (valueDes == null) continue;
          result.allowAppDeletion = valueDes;
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



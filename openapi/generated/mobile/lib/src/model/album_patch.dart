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
/// * [description] - The album's free-text description; the value replaces the stored text wholesale (an empty string clears it). At most 10,000 Unicode characters. 
/// * [coverAssetId] - The pinned cover asset. When present it must be a LIVE member of THIS album (the membership row exists and the asset is not trashed, not hidden, not a Live Photo component row) -- otherwise the answer is 400 with the message key `album_cover_not_member`. Absent (or JSON null, which decodes as absent) leaves the pin unchanged. 
/// * [locked] - The owner's lock. `true` freezes the album's curation and property fields against later writes (this same endpoint refuses them with 409 until the owner patches `locked: false` -- the unlock, never gated itself). Locking does not gate the album's delete. 
@BuiltValue()
abstract class AlbumPatch implements Built<AlbumPatch, AlbumPatchBuilder> {
  /// The album's display title, trimmed server-side; when present it must span 1-64 Unicode characters after the trim. 
  @BuiltValueField(wireName: r'title')
  String? get title;

  /// The album's free-text description; the value replaces the stored text wholesale (an empty string clears it). At most 10,000 Unicode characters. 
  @BuiltValueField(wireName: r'description')
  String? get description;

  /// The pinned cover asset. When present it must be a LIVE member of THIS album (the membership row exists and the asset is not trashed, not hidden, not a Live Photo component row) -- otherwise the answer is 400 with the message key `album_cover_not_member`. Absent (or JSON null, which decodes as absent) leaves the pin unchanged. 
  @BuiltValueField(wireName: r'coverAssetId')
  String? get coverAssetId;

  /// The owner's lock. `true` freezes the album's curation and property fields against later writes (this same endpoint refuses them with 409 until the owner patches `locked: false` -- the unlock, never gated itself). Locking does not gate the album's delete. 
  @BuiltValueField(wireName: r'locked')
  bool? get locked;

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
    if (object.description != null) {
      yield r'description';
      yield serializers.serialize(
        object.description,
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
    if (object.locked != null) {
      yield r'locked';
      yield serializers.serialize(
        object.locked,
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
        case r'description':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.description = valueDes;
          break;
        case r'coverAssetId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.coverAssetId = valueDes;
          break;
        case r'locked':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(bool),
          ) as bool?;
          if (valueDes == null) continue;
          result.locked = valueDes;
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



//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/exif_summary.dart';
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'asset_detail.g.dart';

/// One asset as the detail endpoint returns it: the upload response's identity and storage fields, the pipeline's refinements, and the two owner-level attributes the PATCH endpoint edits (`description`, `isHidden`). The shape is fixed -- every property is always present -- and the pipeline-dependent values (`width`, `height`, `durationMs`, `blurhash` and the fields inside `exif`) are null until the jobs produced them, turning non-null when `asset.ready` fired. A value that stays null on a ready asset means the file carries no such metadata. 
///
/// Properties:
/// * [id] - UUIDv7 identifier of the asset.
/// * [ownerId] - UUIDv7 identifier of the owning account.
/// * [kind] - Media family of the asset. A Live Photo is an `image` whose video component is stored as a linked `video` row that never appears in the timeline. 
/// * [originalFilename] - The file name the asset was uploaded with; the original download re-uses it (RFC 5987 escaped) for `Content-Disposition`. 
/// * [fileSize] - Size of the stored file in bytes.
/// * [mimeType] - Canonical content type of the stored format, for example `image/jpeg` or `video/quicktime`. 
/// * [checksum] - SHA-256 digest of the file bytes, lowercase hex. The prefix of the entity tags the media endpoints serve. 
/// * [createdAt] - When the asset record was created (RFC 3339 UTC).
/// * [takenAt] - Capture time (RFC 3339). Seeded from `deviceCreatedAt` (or the upload time when absent) and refined from EXIF by the metadata pipeline; the response renders the best value the server has at answer time. 
/// * [takenAtSource] - Where `takenAt` comes from: `import` is the device-reported capture time of the upload, `exif` an EXIF capture time the metadata job resolved with an offset, `file` a file timestamp (reserved for the future directory-album importer). 
/// * [isTrashed] - True when the asset sits in the trash.
/// * [isHidden] - True while the owner-level visibility switch is off: the asset is out of every live view (the timeline, the favorites) and manageable only through the hidden view, until it is unhidden. The same switch `PATCH /assets/{id}` and the batch `hide` / `unhide` actions flip. 
/// * [description] - The owner's free-text description of the asset, the text the detail panel renders; the empty string until the owner wrote one. Editable only by the owner through `PATCH /assets/{id}`, which replaces it wholesale. 
/// * [isReady] - True once the whole pipeline of the asset completed -- the metadata and thumbnail jobs for an image, plus the transcode job for a video (for a Live Photo component, the transcode job alone); the `asset.ready` event fires at that moment. Until then the client shows placeholders and the media endpoints answer `ASSET_NOT_READY` for missing derived files. 
/// * [width] - Pixel width in sensor order (not orientation-corrected); null until the metadata job refined the dimensions. 
/// * [height] - Pixel height in sensor order (not orientation-corrected); null until the metadata job refined the dimensions. 
/// * [durationMs] - Playback duration of a video in milliseconds; null for images and until the metadata job refined it. 
/// * [blurhash] - The blurhash of the tile (image assets) or poster frame (video assets) for placeholder rendering; null until the thumbnail job stored it. 
/// * [livePhotoVideoId] - For a Live Photo image, the id of its linked component video row (never in the timeline; its original is downloadable by that id, and its transcoded play stream is served there too); null otherwise. 
/// * [exif] - The EXIF hot fields. Always present as an object; every field inside it is null until the metadata job wrote the extraction. 
@BuiltValue()
abstract class AssetDetail implements Built<AssetDetail, AssetDetailBuilder> {
  /// UUIDv7 identifier of the asset.
  @BuiltValueField(wireName: r'id')
  String get id;

  /// UUIDv7 identifier of the owning account.
  @BuiltValueField(wireName: r'ownerId')
  String get ownerId;

  /// Media family of the asset. A Live Photo is an `image` whose video component is stored as a linked `video` row that never appears in the timeline. 
  @BuiltValueField(wireName: r'kind')
  AssetDetailKindEnum get kind;
  // enum kindEnum {  image,  video,  };

  /// The file name the asset was uploaded with; the original download re-uses it (RFC 5987 escaped) for `Content-Disposition`. 
  @BuiltValueField(wireName: r'originalFilename')
  String get originalFilename;

  /// Size of the stored file in bytes.
  @BuiltValueField(wireName: r'fileSize')
  int get fileSize;

  /// Canonical content type of the stored format, for example `image/jpeg` or `video/quicktime`. 
  @BuiltValueField(wireName: r'mimeType')
  String get mimeType;

  /// SHA-256 digest of the file bytes, lowercase hex. The prefix of the entity tags the media endpoints serve. 
  @BuiltValueField(wireName: r'checksum')
  String get checksum;

  /// When the asset record was created (RFC 3339 UTC).
  @BuiltValueField(wireName: r'createdAt')
  DateTime get createdAt;

  /// Capture time (RFC 3339). Seeded from `deviceCreatedAt` (or the upload time when absent) and refined from EXIF by the metadata pipeline; the response renders the best value the server has at answer time. 
  @BuiltValueField(wireName: r'takenAt')
  DateTime get takenAt;

  /// Where `takenAt` comes from: `import` is the device-reported capture time of the upload, `exif` an EXIF capture time the metadata job resolved with an offset, `file` a file timestamp (reserved for the future directory-album importer). 
  @BuiltValueField(wireName: r'takenAtSource')
  AssetDetailTakenAtSourceEnum get takenAtSource;
  // enum takenAtSourceEnum {  exif,  file,  import,  };

  /// True when the asset sits in the trash.
  @BuiltValueField(wireName: r'isTrashed')
  bool get isTrashed;

  /// True while the owner-level visibility switch is off: the asset is out of every live view (the timeline, the favorites) and manageable only through the hidden view, until it is unhidden. The same switch `PATCH /assets/{id}` and the batch `hide` / `unhide` actions flip. 
  @BuiltValueField(wireName: r'isHidden')
  bool get isHidden;

  /// The owner's free-text description of the asset, the text the detail panel renders; the empty string until the owner wrote one. Editable only by the owner through `PATCH /assets/{id}`, which replaces it wholesale. 
  @BuiltValueField(wireName: r'description')
  String get description;

  /// True once the whole pipeline of the asset completed -- the metadata and thumbnail jobs for an image, plus the transcode job for a video (for a Live Photo component, the transcode job alone); the `asset.ready` event fires at that moment. Until then the client shows placeholders and the media endpoints answer `ASSET_NOT_READY` for missing derived files. 
  @BuiltValueField(wireName: r'isReady')
  bool get isReady;

  /// Pixel width in sensor order (not orientation-corrected); null until the metadata job refined the dimensions. 
  @BuiltValueField(wireName: r'width')
  int? get width;

  /// Pixel height in sensor order (not orientation-corrected); null until the metadata job refined the dimensions. 
  @BuiltValueField(wireName: r'height')
  int? get height;

  /// Playback duration of a video in milliseconds; null for images and until the metadata job refined it. 
  @BuiltValueField(wireName: r'durationMs')
  int? get durationMs;

  /// The blurhash of the tile (image assets) or poster frame (video assets) for placeholder rendering; null until the thumbnail job stored it. 
  @BuiltValueField(wireName: r'blurhash')
  String? get blurhash;

  /// For a Live Photo image, the id of its linked component video row (never in the timeline; its original is downloadable by that id, and its transcoded play stream is served there too); null otherwise. 
  @BuiltValueField(wireName: r'livePhotoVideoId')
  String? get livePhotoVideoId;

  /// The EXIF hot fields. Always present as an object; every field inside it is null until the metadata job wrote the extraction. 
  @BuiltValueField(wireName: r'exif')
  ExifSummary get exif;

  AssetDetail._();

  factory AssetDetail([void updates(AssetDetailBuilder b)]) = _$AssetDetail;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AssetDetailBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AssetDetail> get serializer => _$AssetDetailSerializer();
}

class _$AssetDetailSerializer implements PrimitiveSerializer<AssetDetail> {
  @override
  final Iterable<Type> types = const [AssetDetail, _$AssetDetail];

  @override
  final String wireName = r'AssetDetail';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AssetDetail object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'ownerId';
    yield serializers.serialize(
      object.ownerId,
      specifiedType: const FullType(String),
    );
    yield r'kind';
    yield serializers.serialize(
      object.kind,
      specifiedType: const FullType(AssetDetailKindEnum),
    );
    yield r'originalFilename';
    yield serializers.serialize(
      object.originalFilename,
      specifiedType: const FullType(String),
    );
    yield r'fileSize';
    yield serializers.serialize(
      object.fileSize,
      specifiedType: const FullType(int),
    );
    yield r'mimeType';
    yield serializers.serialize(
      object.mimeType,
      specifiedType: const FullType(String),
    );
    yield r'checksum';
    yield serializers.serialize(
      object.checksum,
      specifiedType: const FullType(String),
    );
    yield r'createdAt';
    yield serializers.serialize(
      object.createdAt,
      specifiedType: const FullType(DateTime),
    );
    yield r'takenAt';
    yield serializers.serialize(
      object.takenAt,
      specifiedType: const FullType(DateTime),
    );
    yield r'takenAtSource';
    yield serializers.serialize(
      object.takenAtSource,
      specifiedType: const FullType(AssetDetailTakenAtSourceEnum),
    );
    yield r'isTrashed';
    yield serializers.serialize(
      object.isTrashed,
      specifiedType: const FullType(bool),
    );
    yield r'isHidden';
    yield serializers.serialize(
      object.isHidden,
      specifiedType: const FullType(bool),
    );
    yield r'description';
    yield serializers.serialize(
      object.description,
      specifiedType: const FullType(String),
    );
    yield r'isReady';
    yield serializers.serialize(
      object.isReady,
      specifiedType: const FullType(bool),
    );
    yield r'width';
    yield object.width == null ? null : serializers.serialize(
      object.width,
      specifiedType: const FullType.nullable(int),
    );
    yield r'height';
    yield object.height == null ? null : serializers.serialize(
      object.height,
      specifiedType: const FullType.nullable(int),
    );
    yield r'durationMs';
    yield object.durationMs == null ? null : serializers.serialize(
      object.durationMs,
      specifiedType: const FullType.nullable(int),
    );
    yield r'blurhash';
    yield object.blurhash == null ? null : serializers.serialize(
      object.blurhash,
      specifiedType: const FullType.nullable(String),
    );
    yield r'livePhotoVideoId';
    yield object.livePhotoVideoId == null ? null : serializers.serialize(
      object.livePhotoVideoId,
      specifiedType: const FullType.nullable(String),
    );
    yield r'exif';
    yield serializers.serialize(
      object.exif,
      specifiedType: const FullType(ExifSummary),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AssetDetail object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AssetDetailBuilder result,
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
        case r'ownerId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.ownerId = valueDes;
          break;
        case r'kind':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(AssetDetailKindEnum),
          ) as AssetDetailKindEnum;
          result.kind = valueDes;
          break;
        case r'originalFilename':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.originalFilename = valueDes;
          break;
        case r'fileSize':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.fileSize = valueDes;
          break;
        case r'mimeType':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.mimeType = valueDes;
          break;
        case r'checksum':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.checksum = valueDes;
          break;
        case r'createdAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DateTime),
          ) as DateTime;
          result.createdAt = valueDes;
          break;
        case r'takenAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DateTime),
          ) as DateTime;
          result.takenAt = valueDes;
          break;
        case r'takenAtSource':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(AssetDetailTakenAtSourceEnum),
          ) as AssetDetailTakenAtSourceEnum;
          result.takenAtSource = valueDes;
          break;
        case r'isTrashed':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.isTrashed = valueDes;
          break;
        case r'isHidden':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.isHidden = valueDes;
          break;
        case r'description':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.description = valueDes;
          break;
        case r'isReady':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.isReady = valueDes;
          break;
        case r'width':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(int),
          ) as int?;
          if (valueDes == null) continue;
          result.width = valueDes;
          break;
        case r'height':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(int),
          ) as int?;
          if (valueDes == null) continue;
          result.height = valueDes;
          break;
        case r'durationMs':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(int),
          ) as int?;
          if (valueDes == null) continue;
          result.durationMs = valueDes;
          break;
        case r'blurhash':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.blurhash = valueDes;
          break;
        case r'livePhotoVideoId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.livePhotoVideoId = valueDes;
          break;
        case r'exif':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(ExifSummary),
          ) as ExifSummary;
          result.exif.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AssetDetail deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AssetDetailBuilder();
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


/// Media family of the asset. A Live Photo is an `image` whose video component is stored as a linked `video` row that never appears in the timeline. 
class AssetDetailKindEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'image')
  static const AssetDetailKindEnum image = _$assetDetailKindEnum_image;
  @BuiltValueEnumConst(wireName: r'video')
  static const AssetDetailKindEnum video = _$assetDetailKindEnum_video;

  static Serializer<AssetDetailKindEnum> get serializer => _$assetDetailKindEnumSerializer;

  const AssetDetailKindEnum._(String name): super(name);

  static BuiltSet<AssetDetailKindEnum> get values => _$assetDetailKindEnumValues;
  static AssetDetailKindEnum valueOf(String name) => _$assetDetailKindEnumValueOf(name);
}

/// Where `takenAt` comes from: `import` is the device-reported capture time of the upload, `exif` an EXIF capture time the metadata job resolved with an offset, `file` a file timestamp (reserved for the future directory-album importer). 
class AssetDetailTakenAtSourceEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'exif')
  static const AssetDetailTakenAtSourceEnum exif = _$assetDetailTakenAtSourceEnum_exif;
  @BuiltValueEnumConst(wireName: r'file')
  static const AssetDetailTakenAtSourceEnum file = _$assetDetailTakenAtSourceEnum_file;
  @BuiltValueEnumConst(wireName: r'import')
  static const AssetDetailTakenAtSourceEnum import_ = _$assetDetailTakenAtSourceEnum_import_;

  static Serializer<AssetDetailTakenAtSourceEnum> get serializer => _$assetDetailTakenAtSourceEnumSerializer;

  const AssetDetailTakenAtSourceEnum._(String name): super(name);

  static BuiltSet<AssetDetailTakenAtSourceEnum> get values => _$assetDetailTakenAtSourceEnumValues;
  static AssetDetailTakenAtSourceEnum valueOf(String name) => _$assetDetailTakenAtSourceEnumValueOf(name);
}


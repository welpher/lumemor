//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'timeline_item.g.dart';

/// One asset of the timeline: the asset detail's fields minus the `exif` object (the detail endpoint renders the EXIF panel; the timeline renders tiles). Every item is live by construction -- hidden assets, trashed assets and the component video rows of Live Photos never enter the timeline -- so `isTrashed` is always false here and is carried only to keep the item shape identical to the detail's. `isReady` still varies: a fresh upload enters the timeline immediately with its pipeline-dependent fields null, and the client upgrades the tile when `asset.ready` fires. The shape is fixed -- every property is always present. Favorites are per-user state (`isFavorite`) and join this shape with the favorites API of the same milestone. 
///
/// Properties:
/// * [id] - UUIDv7 identifier of the asset.
/// * [ownerId] - UUIDv7 identifier of the owning account.
/// * [kind] - Media family of the asset. A Live Photo is an `image` whose linked component video row never appears in the timeline. 
/// * [originalFilename] - The file name the asset was uploaded with.
/// * [fileSize] - Size of the stored file in bytes.
/// * [mimeType] - Canonical content type of the stored format, for example `image/jpeg` or `video/quicktime`. 
/// * [checksum] - SHA-256 digest of the file bytes, lowercase hex. The prefix of the entity tags the media endpoints serve. 
/// * [createdAt] - When the asset record was created (RFC 3339 UTC).
/// * [takenAt] - Capture time (RFC 3339). Seeded from `deviceCreatedAt` (or the upload time when absent) and refined from EXIF by the metadata pipeline; the timeline orders by it. 
/// * [takenAtSource] - Where `takenAt` comes from: `import` is the device-reported capture time of the upload, `exif` an EXIF capture time the metadata job resolved with an offset, `file` a file timestamp (reserved for the future directory-album importer). 
/// * [isTrashed] - Always false in the timeline: trashed assets never enter it. The field exists to keep the shape identical to the asset detail's. 
/// * [isReady] - True once the whole pipeline of the asset (metadata and serving copies) completed; until then the pipeline-dependent fields are null and the client renders placeholders. 
/// * [width] - Pixel width in sensor order; null until the metadata job refined the dimensions. 
/// * [height] - Pixel height in sensor order; null until the metadata job refined the dimensions. 
/// * [durationMs] - Playback duration of a video in milliseconds; null for images and until the metadata job refined it. 
/// * [blurhash] - The blurhash of the tile (image assets) or poster frame (video assets) for placeholder rendering; null until the thumbnail job stored it. 
/// * [livePhotoVideoId] - For a Live Photo image, the id of its linked component video row (resolvable for playback); null otherwise. 
@BuiltValue()
abstract class TimelineItem implements Built<TimelineItem, TimelineItemBuilder> {
  /// UUIDv7 identifier of the asset.
  @BuiltValueField(wireName: r'id')
  String get id;

  /// UUIDv7 identifier of the owning account.
  @BuiltValueField(wireName: r'ownerId')
  String get ownerId;

  /// Media family of the asset. A Live Photo is an `image` whose linked component video row never appears in the timeline. 
  @BuiltValueField(wireName: r'kind')
  TimelineItemKindEnum get kind;
  // enum kindEnum {  image,  video,  };

  /// The file name the asset was uploaded with.
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

  /// Capture time (RFC 3339). Seeded from `deviceCreatedAt` (or the upload time when absent) and refined from EXIF by the metadata pipeline; the timeline orders by it. 
  @BuiltValueField(wireName: r'takenAt')
  DateTime get takenAt;

  /// Where `takenAt` comes from: `import` is the device-reported capture time of the upload, `exif` an EXIF capture time the metadata job resolved with an offset, `file` a file timestamp (reserved for the future directory-album importer). 
  @BuiltValueField(wireName: r'takenAtSource')
  TimelineItemTakenAtSourceEnum get takenAtSource;
  // enum takenAtSourceEnum {  exif,  file,  import,  };

  /// Always false in the timeline: trashed assets never enter it. The field exists to keep the shape identical to the asset detail's. 
  @BuiltValueField(wireName: r'isTrashed')
  bool get isTrashed;

  /// True once the whole pipeline of the asset (metadata and serving copies) completed; until then the pipeline-dependent fields are null and the client renders placeholders. 
  @BuiltValueField(wireName: r'isReady')
  bool get isReady;

  /// Pixel width in sensor order; null until the metadata job refined the dimensions. 
  @BuiltValueField(wireName: r'width')
  int? get width;

  /// Pixel height in sensor order; null until the metadata job refined the dimensions. 
  @BuiltValueField(wireName: r'height')
  int? get height;

  /// Playback duration of a video in milliseconds; null for images and until the metadata job refined it. 
  @BuiltValueField(wireName: r'durationMs')
  int? get durationMs;

  /// The blurhash of the tile (image assets) or poster frame (video assets) for placeholder rendering; null until the thumbnail job stored it. 
  @BuiltValueField(wireName: r'blurhash')
  String? get blurhash;

  /// For a Live Photo image, the id of its linked component video row (resolvable for playback); null otherwise. 
  @BuiltValueField(wireName: r'livePhotoVideoId')
  String? get livePhotoVideoId;

  TimelineItem._();

  factory TimelineItem([void updates(TimelineItemBuilder b)]) = _$TimelineItem;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TimelineItemBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TimelineItem> get serializer => _$TimelineItemSerializer();
}

class _$TimelineItemSerializer implements PrimitiveSerializer<TimelineItem> {
  @override
  final Iterable<Type> types = const [TimelineItem, _$TimelineItem];

  @override
  final String wireName = r'TimelineItem';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TimelineItem object, {
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
      specifiedType: const FullType(TimelineItemKindEnum),
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
      specifiedType: const FullType(TimelineItemTakenAtSourceEnum),
    );
    yield r'isTrashed';
    yield serializers.serialize(
      object.isTrashed,
      specifiedType: const FullType(bool),
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
  }

  @override
  Object serialize(
    Serializers serializers,
    TimelineItem object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TimelineItemBuilder result,
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
            specifiedType: const FullType(TimelineItemKindEnum),
          ) as TimelineItemKindEnum;
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
            specifiedType: const FullType(TimelineItemTakenAtSourceEnum),
          ) as TimelineItemTakenAtSourceEnum;
          result.takenAtSource = valueDes;
          break;
        case r'isTrashed':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.isTrashed = valueDes;
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
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  TimelineItem deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TimelineItemBuilder();
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


/// Media family of the asset. A Live Photo is an `image` whose linked component video row never appears in the timeline. 
class TimelineItemKindEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'image')
  static const TimelineItemKindEnum image = _$timelineItemKindEnum_image;
  @BuiltValueEnumConst(wireName: r'video')
  static const TimelineItemKindEnum video = _$timelineItemKindEnum_video;

  static Serializer<TimelineItemKindEnum> get serializer => _$timelineItemKindEnumSerializer;

  const TimelineItemKindEnum._(String name): super(name);

  static BuiltSet<TimelineItemKindEnum> get values => _$timelineItemKindEnumValues;
  static TimelineItemKindEnum valueOf(String name) => _$timelineItemKindEnumValueOf(name);
}

/// Where `takenAt` comes from: `import` is the device-reported capture time of the upload, `exif` an EXIF capture time the metadata job resolved with an offset, `file` a file timestamp (reserved for the future directory-album importer). 
class TimelineItemTakenAtSourceEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'exif')
  static const TimelineItemTakenAtSourceEnum exif = _$timelineItemTakenAtSourceEnum_exif;
  @BuiltValueEnumConst(wireName: r'file')
  static const TimelineItemTakenAtSourceEnum file = _$timelineItemTakenAtSourceEnum_file;
  @BuiltValueEnumConst(wireName: r'import')
  static const TimelineItemTakenAtSourceEnum import_ = _$timelineItemTakenAtSourceEnum_import_;

  static Serializer<TimelineItemTakenAtSourceEnum> get serializer => _$timelineItemTakenAtSourceEnumSerializer;

  const TimelineItemTakenAtSourceEnum._(String name): super(name);

  static BuiltSet<TimelineItemTakenAtSourceEnum> get values => _$timelineItemTakenAtSourceEnumValues;
  static TimelineItemTakenAtSourceEnum valueOf(String name) => _$timelineItemTakenAtSourceEnumValueOf(name);
}


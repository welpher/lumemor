//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'trash_item.g.dart';

/// One asset of the trash view: the timeline item's fields (the tile-relevant identity and pipeline fields, so the trash page renders the same tiles as the timeline) plus the trash semantics -- `deletedAt`, when the soft delete happened, and `remainingDays`, the countdown of whole days until the expiry job removes the row (rounded up; 0 once past the deadline, for the short window until the hourly job catches up). Every item is trashed by construction; there is no `isTrashed` field. The component video rows of Live Photos never appear: the pair is represented by its photo. The shape is fixed -- every property is always present. 
///
/// Properties:
/// * [id] - UUIDv7 identifier of the asset.
/// * [ownerId] - UUIDv7 identifier of the owning account.
/// * [kind] - Media family of the asset.
/// * [originalFilename] - The file name the asset was uploaded with.
/// * [fileSize] - Size of the stored file in bytes.
/// * [mimeType] - Canonical content type of the stored format, for example `image/jpeg` or `video/quicktime`. 
/// * [checksum] - SHA-256 digest of the file bytes, lowercase hex. The prefix of the entity tags the media endpoints serve. 
/// * [createdAt] - When the asset record was created (RFC 3339 UTC).
/// * [takenAt] - Capture time (RFC 3339), the value the timeline would order the asset by. 
/// * [takenAtSource] - Where `takenAt` comes from.
/// * [isReady] - True once the whole pipeline of the asset (metadata and serving copies) completed. 
/// * [width] - Pixel width in sensor order; null until the metadata job refined the dimensions. 
/// * [height] - Pixel height in sensor order; null until the metadata job refined the dimensions. 
/// * [durationMs] - Playback duration of a video in milliseconds; null for images and until the metadata job refined it. 
/// * [blurhash] - The blurhash of the tile for placeholder rendering; null until the thumbnail job stored it. 
/// * [livePhotoVideoId] - For a Live Photo image, the id of its linked component video row (restored and expired together with the photo); null otherwise. 
/// * [deletedAt] - When the asset was soft deleted (RFC 3339 UTC). The retention window ends at `deletedAt` plus the instance's `trash_retention_days`. 
/// * [remainingDays] - Whole days until the expiry job removes the asset, rounded up (an asset trashed moments ago with the default retention of 30 days shows 30). 0 once the deadline has passed -- the row survives only until the next hourly expiry run. 
@BuiltValue()
abstract class TrashItem implements Built<TrashItem, TrashItemBuilder> {
  /// UUIDv7 identifier of the asset.
  @BuiltValueField(wireName: r'id')
  String get id;

  /// UUIDv7 identifier of the owning account.
  @BuiltValueField(wireName: r'ownerId')
  String get ownerId;

  /// Media family of the asset.
  @BuiltValueField(wireName: r'kind')
  TrashItemKindEnum get kind;
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

  /// Capture time (RFC 3339), the value the timeline would order the asset by. 
  @BuiltValueField(wireName: r'takenAt')
  DateTime get takenAt;

  /// Where `takenAt` comes from.
  @BuiltValueField(wireName: r'takenAtSource')
  TrashItemTakenAtSourceEnum get takenAtSource;
  // enum takenAtSourceEnum {  exif,  file,  import,  };

  /// True once the whole pipeline of the asset (metadata and serving copies) completed. 
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

  /// The blurhash of the tile for placeholder rendering; null until the thumbnail job stored it. 
  @BuiltValueField(wireName: r'blurhash')
  String? get blurhash;

  /// For a Live Photo image, the id of its linked component video row (restored and expired together with the photo); null otherwise. 
  @BuiltValueField(wireName: r'livePhotoVideoId')
  String? get livePhotoVideoId;

  /// When the asset was soft deleted (RFC 3339 UTC). The retention window ends at `deletedAt` plus the instance's `trash_retention_days`. 
  @BuiltValueField(wireName: r'deletedAt')
  DateTime get deletedAt;

  /// Whole days until the expiry job removes the asset, rounded up (an asset trashed moments ago with the default retention of 30 days shows 30). 0 once the deadline has passed -- the row survives only until the next hourly expiry run. 
  @BuiltValueField(wireName: r'remainingDays')
  int get remainingDays;

  TrashItem._();

  factory TrashItem([void updates(TrashItemBuilder b)]) = _$TrashItem;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TrashItemBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TrashItem> get serializer => _$TrashItemSerializer();
}

class _$TrashItemSerializer implements PrimitiveSerializer<TrashItem> {
  @override
  final Iterable<Type> types = const [TrashItem, _$TrashItem];

  @override
  final String wireName = r'TrashItem';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TrashItem object, {
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
      specifiedType: const FullType(TrashItemKindEnum),
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
      specifiedType: const FullType(TrashItemTakenAtSourceEnum),
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
    yield r'deletedAt';
    yield serializers.serialize(
      object.deletedAt,
      specifiedType: const FullType(DateTime),
    );
    yield r'remainingDays';
    yield serializers.serialize(
      object.remainingDays,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    TrashItem object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TrashItemBuilder result,
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
            specifiedType: const FullType(TrashItemKindEnum),
          ) as TrashItemKindEnum;
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
            specifiedType: const FullType(TrashItemTakenAtSourceEnum),
          ) as TrashItemTakenAtSourceEnum;
          result.takenAtSource = valueDes;
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
        case r'deletedAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DateTime),
          ) as DateTime;
          result.deletedAt = valueDes;
          break;
        case r'remainingDays':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.remainingDays = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  TrashItem deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TrashItemBuilder();
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


/// Media family of the asset.
class TrashItemKindEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'image')
  static const TrashItemKindEnum image = _$trashItemKindEnum_image;
  @BuiltValueEnumConst(wireName: r'video')
  static const TrashItemKindEnum video = _$trashItemKindEnum_video;

  static Serializer<TrashItemKindEnum> get serializer => _$trashItemKindEnumSerializer;

  const TrashItemKindEnum._(String name): super(name);

  static BuiltSet<TrashItemKindEnum> get values => _$trashItemKindEnumValues;
  static TrashItemKindEnum valueOf(String name) => _$trashItemKindEnumValueOf(name);
}

/// Where `takenAt` comes from.
class TrashItemTakenAtSourceEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'exif')
  static const TrashItemTakenAtSourceEnum exif = _$trashItemTakenAtSourceEnum_exif;
  @BuiltValueEnumConst(wireName: r'file')
  static const TrashItemTakenAtSourceEnum file = _$trashItemTakenAtSourceEnum_file;
  @BuiltValueEnumConst(wireName: r'import')
  static const TrashItemTakenAtSourceEnum import_ = _$trashItemTakenAtSourceEnum_import_;

  static Serializer<TrashItemTakenAtSourceEnum> get serializer => _$trashItemTakenAtSourceEnumSerializer;

  const TrashItemTakenAtSourceEnum._(String name): super(name);

  static BuiltSet<TrashItemTakenAtSourceEnum> get values => _$trashItemTakenAtSourceEnumValues;
  static TrashItemTakenAtSourceEnum valueOf(String name) => _$trashItemTakenAtSourceEnumValueOf(name);
}


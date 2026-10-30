//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'asset_response.g.dart';

/// One stored asset as the upload and idempotency endpoints return it. `state` discriminates the upload outcome: `created` (201, a new asset), `duplicate` (200, an identical live asset already existed), `trashed` (200, an identical asset sits in the trash; the body carries that asset so the client can offer restore or save-as-new). The idempotency lookup mirrors the same states: it reports what a re-upload of the content would yield, so `created` never appears there. 
///
/// Properties:
/// * [id] - UUIDv7 identifier of the asset.
/// * [ownerId] - UUIDv7 identifier of the owning account.
/// * [kind] - Media family of the asset. A Live Photo is an `image` whose video component is stored as a linked `video` row that never appears in the timeline. 
/// * [originalFilename] - The file name the asset was uploaded with: the client-side `fileName` of the photo (the component video keeps its own file part name). 
/// * [fileSize] - Size of the stored file in bytes.
/// * [mimeType] - Canonical content type of the stored format, for example `image/jpeg` or `video/quicktime`. 
/// * [checksum] - SHA-256 digest of the file bytes, lowercase hex. The owner-scoped idempotency key of the upload. 
/// * [state] - The upload outcome: `created` (201), `duplicate` (200, an identical live asset) or `trashed` (200, an identical asset in the trash). 
/// * [createdAt] - When the asset record was created (RFC 3339 UTC).
/// * [takenAt] - Capture time (RFC 3339). Seeded from `deviceCreatedAt` (or the upload time when absent) and refined from EXIF by the metadata pipeline; responses render the best value the server has at answer time. 
/// * [isTrashed] - True when the asset sits in the trash; `state: trashed` responses carry it as true so clients can offer the restore or save-as-new decision. 
@BuiltValue()
abstract class AssetResponse implements Built<AssetResponse, AssetResponseBuilder> {
  /// UUIDv7 identifier of the asset.
  @BuiltValueField(wireName: r'id')
  String get id;

  /// UUIDv7 identifier of the owning account.
  @BuiltValueField(wireName: r'ownerId')
  String get ownerId;

  /// Media family of the asset. A Live Photo is an `image` whose video component is stored as a linked `video` row that never appears in the timeline. 
  @BuiltValueField(wireName: r'kind')
  AssetResponseKindEnum get kind;
  // enum kindEnum {  image,  video,  };

  /// The file name the asset was uploaded with: the client-side `fileName` of the photo (the component video keeps its own file part name). 
  @BuiltValueField(wireName: r'originalFilename')
  String get originalFilename;

  /// Size of the stored file in bytes.
  @BuiltValueField(wireName: r'fileSize')
  int get fileSize;

  /// Canonical content type of the stored format, for example `image/jpeg` or `video/quicktime`. 
  @BuiltValueField(wireName: r'mimeType')
  String get mimeType;

  /// SHA-256 digest of the file bytes, lowercase hex. The owner-scoped idempotency key of the upload. 
  @BuiltValueField(wireName: r'checksum')
  String get checksum;

  /// The upload outcome: `created` (201), `duplicate` (200, an identical live asset) or `trashed` (200, an identical asset in the trash). 
  @BuiltValueField(wireName: r'state')
  AssetResponseStateEnum get state;
  // enum stateEnum {  created,  duplicate,  trashed,  };

  /// When the asset record was created (RFC 3339 UTC).
  @BuiltValueField(wireName: r'createdAt')
  DateTime get createdAt;

  /// Capture time (RFC 3339). Seeded from `deviceCreatedAt` (or the upload time when absent) and refined from EXIF by the metadata pipeline; responses render the best value the server has at answer time. 
  @BuiltValueField(wireName: r'takenAt')
  DateTime get takenAt;

  /// True when the asset sits in the trash; `state: trashed` responses carry it as true so clients can offer the restore or save-as-new decision. 
  @BuiltValueField(wireName: r'isTrashed')
  bool get isTrashed;

  AssetResponse._();

  factory AssetResponse([void updates(AssetResponseBuilder b)]) = _$AssetResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AssetResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AssetResponse> get serializer => _$AssetResponseSerializer();
}

class _$AssetResponseSerializer implements PrimitiveSerializer<AssetResponse> {
  @override
  final Iterable<Type> types = const [AssetResponse, _$AssetResponse];

  @override
  final String wireName = r'AssetResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AssetResponse object, {
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
      specifiedType: const FullType(AssetResponseKindEnum),
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
    yield r'state';
    yield serializers.serialize(
      object.state,
      specifiedType: const FullType(AssetResponseStateEnum),
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
    yield r'isTrashed';
    yield serializers.serialize(
      object.isTrashed,
      specifiedType: const FullType(bool),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AssetResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AssetResponseBuilder result,
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
            specifiedType: const FullType(AssetResponseKindEnum),
          ) as AssetResponseKindEnum;
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
        case r'state':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(AssetResponseStateEnum),
          ) as AssetResponseStateEnum;
          result.state = valueDes;
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
        case r'isTrashed':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.isTrashed = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AssetResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AssetResponseBuilder();
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
class AssetResponseKindEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'image')
  static const AssetResponseKindEnum image = _$assetResponseKindEnum_image;
  @BuiltValueEnumConst(wireName: r'video')
  static const AssetResponseKindEnum video = _$assetResponseKindEnum_video;

  static Serializer<AssetResponseKindEnum> get serializer => _$assetResponseKindEnumSerializer;

  const AssetResponseKindEnum._(String name): super(name);

  static BuiltSet<AssetResponseKindEnum> get values => _$assetResponseKindEnumValues;
  static AssetResponseKindEnum valueOf(String name) => _$assetResponseKindEnumValueOf(name);
}

/// The upload outcome: `created` (201), `duplicate` (200, an identical live asset) or `trashed` (200, an identical asset in the trash). 
class AssetResponseStateEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'created')
  static const AssetResponseStateEnum created = _$assetResponseStateEnum_created;
  @BuiltValueEnumConst(wireName: r'duplicate')
  static const AssetResponseStateEnum duplicate = _$assetResponseStateEnum_duplicate;
  @BuiltValueEnumConst(wireName: r'trashed')
  static const AssetResponseStateEnum trashed = _$assetResponseStateEnum_trashed;

  static Serializer<AssetResponseStateEnum> get serializer => _$assetResponseStateEnumSerializer;

  const AssetResponseStateEnum._(String name): super(name);

  static BuiltSet<AssetResponseStateEnum> get values => _$assetResponseStateEnumValues;
  static AssetResponseStateEnum valueOf(String name) => _$assetResponseStateEnumValueOf(name);
}


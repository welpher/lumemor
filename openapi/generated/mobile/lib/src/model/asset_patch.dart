//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'asset_patch.g.dart';

/// The partial edit of one asset: every field is optional, at least one must be present (a body that carries none -- null values count as absent -- is a client bug and answers 400), and only the fields the request carries update; the rest keep their values. Every field is owner-level asset state, so a patch that actually changes one of them publishes a single `asset.updated` event; a patch that repeats the stored values is the idempotent no-op and publishes nothing. 
///
/// Properties:
/// * [description] - The owner's free-text description of the asset, the text the detail panel renders. The value replaces the stored text wholesale (an empty string clears it). At most 10,000 Unicode characters -- the count is over runes, so a text of multi-byte characters stays within the bound at the same length. 
/// * [isHidden] - The owner-level visibility switch: `true` keeps the asset out of every live view (the timeline, the favorites) until it is unhidden; `false` returns it. The same switch the batch `hide` / `unhide` actions flip. 
/// * [takenAt] - The owner's manual capture-time edit: the value replaces the stored instant wholesale. The timestamp MUST carry a UTC offset (RFC 3339's `Z` or `+hh:mm` designator) -- the offset is the capture zone the info panel displays beside the time, so a zone-less value is ambiguous and answers 400 `VALIDATION_FAILED`. The same value decides where the photo sits on the timeline (the instant) and which offset renders beside it (the offset). 
/// * [latitude] - The manual location edit's latitude, in degrees (-90..90). It rides with `longitude` (the pair is the location; one half alone answers 400 `VALIDATION_FAILED`). Writing the pair replaces the asset's location: the info panel renders these coordinates and the resolved place follows the pin. The camera-recorded GPS in `exif` keeps the file's own reading and is not touched. 
/// * [longitude] - The manual location edit's longitude, in degrees (-180..180). It rides with `latitude` (see there for the pair's semantics). 
@BuiltValue()
abstract class AssetPatch implements Built<AssetPatch, AssetPatchBuilder> {
  /// The owner's free-text description of the asset, the text the detail panel renders. The value replaces the stored text wholesale (an empty string clears it). At most 10,000 Unicode characters -- the count is over runes, so a text of multi-byte characters stays within the bound at the same length. 
  @BuiltValueField(wireName: r'description')
  String? get description;

  /// The owner-level visibility switch: `true` keeps the asset out of every live view (the timeline, the favorites) until it is unhidden; `false` returns it. The same switch the batch `hide` / `unhide` actions flip. 
  @BuiltValueField(wireName: r'isHidden')
  bool? get isHidden;

  /// The owner's manual capture-time edit: the value replaces the stored instant wholesale. The timestamp MUST carry a UTC offset (RFC 3339's `Z` or `+hh:mm` designator) -- the offset is the capture zone the info panel displays beside the time, so a zone-less value is ambiguous and answers 400 `VALIDATION_FAILED`. The same value decides where the photo sits on the timeline (the instant) and which offset renders beside it (the offset). 
  @BuiltValueField(wireName: r'takenAt')
  DateTime? get takenAt;

  /// The manual location edit's latitude, in degrees (-90..90). It rides with `longitude` (the pair is the location; one half alone answers 400 `VALIDATION_FAILED`). Writing the pair replaces the asset's location: the info panel renders these coordinates and the resolved place follows the pin. The camera-recorded GPS in `exif` keeps the file's own reading and is not touched. 
  @BuiltValueField(wireName: r'latitude')
  double? get latitude;

  /// The manual location edit's longitude, in degrees (-180..180). It rides with `latitude` (see there for the pair's semantics). 
  @BuiltValueField(wireName: r'longitude')
  double? get longitude;

  AssetPatch._();

  factory AssetPatch([void updates(AssetPatchBuilder b)]) = _$AssetPatch;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AssetPatchBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AssetPatch> get serializer => _$AssetPatchSerializer();
}

class _$AssetPatchSerializer implements PrimitiveSerializer<AssetPatch> {
  @override
  final Iterable<Type> types = const [AssetPatch, _$AssetPatch];

  @override
  final String wireName = r'AssetPatch';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AssetPatch object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    if (object.description != null) {
      yield r'description';
      yield serializers.serialize(
        object.description,
        specifiedType: const FullType(String),
      );
    }
    if (object.isHidden != null) {
      yield r'isHidden';
      yield serializers.serialize(
        object.isHidden,
        specifiedType: const FullType(bool),
      );
    }
    if (object.takenAt != null) {
      yield r'takenAt';
      yield serializers.serialize(
        object.takenAt,
        specifiedType: const FullType(DateTime),
      );
    }
    if (object.latitude != null) {
      yield r'latitude';
      yield serializers.serialize(
        object.latitude,
        specifiedType: const FullType(double),
      );
    }
    if (object.longitude != null) {
      yield r'longitude';
      yield serializers.serialize(
        object.longitude,
        specifiedType: const FullType(double),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    AssetPatch object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AssetPatchBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'description':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.description = valueDes;
          break;
        case r'isHidden':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(bool),
          ) as bool?;
          if (valueDes == null) continue;
          result.isHidden = valueDes;
          break;
        case r'takenAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(DateTime),
          ) as DateTime?;
          if (valueDes == null) continue;
          result.takenAt = valueDes;
          break;
        case r'latitude':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(double),
          ) as double?;
          if (valueDes == null) continue;
          result.latitude = valueDes;
          break;
        case r'longitude':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(double),
          ) as double?;
          if (valueDes == null) continue;
          result.longitude = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AssetPatch deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AssetPatchBuilder();
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



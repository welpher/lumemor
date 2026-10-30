//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'exif_summary.g.dart';

/// The EXIF hot fields of an asset, flattened from the raw exiftool output by the metadata job. Every property is always present and null until the job wrote it -- or forever, for files that carry no such tag. 
///
/// Properties:
/// * [cameraMake] - Camera manufacturer (EXIF Make).
/// * [cameraModel] - Camera model (EXIF Model).
/// * [lens] - Lens identifier (LensModel, LensID or Lens).
/// * [iso] - ISO sensitivity.
/// * [aperture] - Aperture as the f-number.
/// * [shutter] - Exposure time in photo notation, for example `1/250` or `2.5s`. 
/// * [focal] - Focal length in millimeters.
/// * [lat] - GPS latitude in degrees.
/// * [lon] - GPS longitude in degrees.
/// * [alt] - GPS altitude in meters.
/// * [tzOffset] - The offset that resolved the capture time, in EXIF notation (`+05:30`); null when the capture time did not resolve through EXIF. 
@BuiltValue()
abstract class ExifSummary implements Built<ExifSummary, ExifSummaryBuilder> {
  /// Camera manufacturer (EXIF Make).
  @BuiltValueField(wireName: r'cameraMake')
  String? get cameraMake;

  /// Camera model (EXIF Model).
  @BuiltValueField(wireName: r'cameraModel')
  String? get cameraModel;

  /// Lens identifier (LensModel, LensID or Lens).
  @BuiltValueField(wireName: r'lens')
  String? get lens;

  /// ISO sensitivity.
  @BuiltValueField(wireName: r'iso')
  int? get iso;

  /// Aperture as the f-number.
  @BuiltValueField(wireName: r'aperture')
  num? get aperture;

  /// Exposure time in photo notation, for example `1/250` or `2.5s`. 
  @BuiltValueField(wireName: r'shutter')
  String? get shutter;

  /// Focal length in millimeters.
  @BuiltValueField(wireName: r'focal')
  num? get focal;

  /// GPS latitude in degrees.
  @BuiltValueField(wireName: r'lat')
  double? get lat;

  /// GPS longitude in degrees.
  @BuiltValueField(wireName: r'lon')
  double? get lon;

  /// GPS altitude in meters.
  @BuiltValueField(wireName: r'alt')
  num? get alt;

  /// The offset that resolved the capture time, in EXIF notation (`+05:30`); null when the capture time did not resolve through EXIF. 
  @BuiltValueField(wireName: r'tzOffset')
  String? get tzOffset;

  ExifSummary._();

  factory ExifSummary([void updates(ExifSummaryBuilder b)]) = _$ExifSummary;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ExifSummaryBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ExifSummary> get serializer => _$ExifSummarySerializer();
}

class _$ExifSummarySerializer implements PrimitiveSerializer<ExifSummary> {
  @override
  final Iterable<Type> types = const [ExifSummary, _$ExifSummary];

  @override
  final String wireName = r'ExifSummary';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ExifSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'cameraMake';
    yield object.cameraMake == null ? null : serializers.serialize(
      object.cameraMake,
      specifiedType: const FullType.nullable(String),
    );
    yield r'cameraModel';
    yield object.cameraModel == null ? null : serializers.serialize(
      object.cameraModel,
      specifiedType: const FullType.nullable(String),
    );
    yield r'lens';
    yield object.lens == null ? null : serializers.serialize(
      object.lens,
      specifiedType: const FullType.nullable(String),
    );
    yield r'iso';
    yield object.iso == null ? null : serializers.serialize(
      object.iso,
      specifiedType: const FullType.nullable(int),
    );
    yield r'aperture';
    yield object.aperture == null ? null : serializers.serialize(
      object.aperture,
      specifiedType: const FullType.nullable(num),
    );
    yield r'shutter';
    yield object.shutter == null ? null : serializers.serialize(
      object.shutter,
      specifiedType: const FullType.nullable(String),
    );
    yield r'focal';
    yield object.focal == null ? null : serializers.serialize(
      object.focal,
      specifiedType: const FullType.nullable(num),
    );
    yield r'lat';
    yield object.lat == null ? null : serializers.serialize(
      object.lat,
      specifiedType: const FullType.nullable(double),
    );
    yield r'lon';
    yield object.lon == null ? null : serializers.serialize(
      object.lon,
      specifiedType: const FullType.nullable(double),
    );
    yield r'alt';
    yield object.alt == null ? null : serializers.serialize(
      object.alt,
      specifiedType: const FullType.nullable(num),
    );
    yield r'tzOffset';
    yield object.tzOffset == null ? null : serializers.serialize(
      object.tzOffset,
      specifiedType: const FullType.nullable(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ExifSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ExifSummaryBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'cameraMake':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.cameraMake = valueDes;
          break;
        case r'cameraModel':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.cameraModel = valueDes;
          break;
        case r'lens':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.lens = valueDes;
          break;
        case r'iso':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(int),
          ) as int?;
          if (valueDes == null) continue;
          result.iso = valueDes;
          break;
        case r'aperture':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(num),
          ) as num?;
          if (valueDes == null) continue;
          result.aperture = valueDes;
          break;
        case r'shutter':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.shutter = valueDes;
          break;
        case r'focal':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(num),
          ) as num?;
          if (valueDes == null) continue;
          result.focal = valueDes;
          break;
        case r'lat':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(double),
          ) as double?;
          if (valueDes == null) continue;
          result.lat = valueDes;
          break;
        case r'lon':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(double),
          ) as double?;
          if (valueDes == null) continue;
          result.lon = valueDes;
          break;
        case r'alt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(num),
          ) as num?;
          if (valueDes == null) continue;
          result.alt = valueDes;
          break;
        case r'tzOffset':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.tzOffset = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ExifSummary deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ExifSummaryBuilder();
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



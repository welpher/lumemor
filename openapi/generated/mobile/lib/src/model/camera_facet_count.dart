//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'camera_facet_count.g.dart';

/// One camera of the cameras facet.
///
/// Properties:
/// * [id] - The round-trip filter value: `make|model` with both halves trimmed, empty where the EXIF row carries no value. 
/// * [make] - The EXIF camera make, null when absent.
/// * [model] - The EXIF camera model, null when absent.
/// * [assetCount] - The number of assets of the current filtered set (minus the camera filter) shot with this camera. 
@BuiltValue()
abstract class CameraFacetCount implements Built<CameraFacetCount, CameraFacetCountBuilder> {
  /// The round-trip filter value: `make|model` with both halves trimmed, empty where the EXIF row carries no value. 
  @BuiltValueField(wireName: r'id')
  String get id;

  /// The EXIF camera make, null when absent.
  @BuiltValueField(wireName: r'make')
  String? get make;

  /// The EXIF camera model, null when absent.
  @BuiltValueField(wireName: r'model')
  String? get model;

  /// The number of assets of the current filtered set (minus the camera filter) shot with this camera. 
  @BuiltValueField(wireName: r'assetCount')
  int get assetCount;

  CameraFacetCount._();

  factory CameraFacetCount([void updates(CameraFacetCountBuilder b)]) = _$CameraFacetCount;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(CameraFacetCountBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<CameraFacetCount> get serializer => _$CameraFacetCountSerializer();
}

class _$CameraFacetCountSerializer implements PrimitiveSerializer<CameraFacetCount> {
  @override
  final Iterable<Type> types = const [CameraFacetCount, _$CameraFacetCount];

  @override
  final String wireName = r'CameraFacetCount';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    CameraFacetCount object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    if (object.make != null) {
      yield r'make';
      yield serializers.serialize(
        object.make,
        specifiedType: const FullType(String),
      );
    }
    if (object.model != null) {
      yield r'model';
      yield serializers.serialize(
        object.model,
        specifiedType: const FullType(String),
      );
    }
    yield r'assetCount';
    yield serializers.serialize(
      object.assetCount,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    CameraFacetCount object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required CameraFacetCountBuilder result,
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
        case r'make':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.make = valueDes;
          break;
        case r'model':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.model = valueDes;
          break;
        case r'assetCount':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.assetCount = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  CameraFacetCount deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = CameraFacetCountBuilder();
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



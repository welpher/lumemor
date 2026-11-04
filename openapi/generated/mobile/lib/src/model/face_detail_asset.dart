//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'face_detail_asset.g.dart';

/// The minimal context of the asset the face sits on: when it was taken and under what file name -- the two lines the cluster detail renders under each crop. 
///
/// Properties:
/// * [takenAt] - When the asset was taken (RFC 3339 UTC).
/// * [originalFilename] - The file name the asset was uploaded under.
@BuiltValue()
abstract class FaceDetailAsset implements Built<FaceDetailAsset, FaceDetailAssetBuilder> {
  /// When the asset was taken (RFC 3339 UTC).
  @BuiltValueField(wireName: r'takenAt')
  DateTime get takenAt;

  /// The file name the asset was uploaded under.
  @BuiltValueField(wireName: r'originalFilename')
  String get originalFilename;

  FaceDetailAsset._();

  factory FaceDetailAsset([void updates(FaceDetailAssetBuilder b)]) = _$FaceDetailAsset;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(FaceDetailAssetBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<FaceDetailAsset> get serializer => _$FaceDetailAssetSerializer();
}

class _$FaceDetailAssetSerializer implements PrimitiveSerializer<FaceDetailAsset> {
  @override
  final Iterable<Type> types = const [FaceDetailAsset, _$FaceDetailAsset];

  @override
  final String wireName = r'FaceDetailAsset';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    FaceDetailAsset object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'takenAt';
    yield serializers.serialize(
      object.takenAt,
      specifiedType: const FullType(DateTime),
    );
    yield r'originalFilename';
    yield serializers.serialize(
      object.originalFilename,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    FaceDetailAsset object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required FaceDetailAssetBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'takenAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DateTime),
          ) as DateTime;
          result.takenAt = valueDes;
          break;
        case r'originalFilename':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.originalFilename = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  FaceDetailAsset deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = FaceDetailAssetBuilder();
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



//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/face_detail_asset.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'face_detail.g.dart';

/// One face of the cluster on an asset visible to the caller: the crop's identity, the detection score and the minimal asset context the detail grid renders. 
///
/// Properties:
/// * [id] - Identifier of the face row.
/// * [assetId] - Identifier of the asset the face sits on.
/// * [score] - The detector's confidence for this face.
/// * [thumbnail] - URL of the face's crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with this face's id, serving the bounding-box crop of the asset's view-size derivative. 
/// * [asset] 
@BuiltValue()
abstract class FaceDetail implements Built<FaceDetail, FaceDetailBuilder> {
  /// Identifier of the face row.
  @BuiltValueField(wireName: r'id')
  String get id;

  /// Identifier of the asset the face sits on.
  @BuiltValueField(wireName: r'assetId')
  String get assetId;

  /// The detector's confidence for this face.
  @BuiltValueField(wireName: r'score')
  double get score;

  /// URL of the face's crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with this face's id, serving the bounding-box crop of the asset's view-size derivative. 
  @BuiltValueField(wireName: r'thumbnail')
  String get thumbnail;

  @BuiltValueField(wireName: r'asset')
  FaceDetailAsset get asset;

  FaceDetail._();

  factory FaceDetail([void updates(FaceDetailBuilder b)]) = _$FaceDetail;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(FaceDetailBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<FaceDetail> get serializer => _$FaceDetailSerializer();
}

class _$FaceDetailSerializer implements PrimitiveSerializer<FaceDetail> {
  @override
  final Iterable<Type> types = const [FaceDetail, _$FaceDetail];

  @override
  final String wireName = r'FaceDetail';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    FaceDetail object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'assetId';
    yield serializers.serialize(
      object.assetId,
      specifiedType: const FullType(String),
    );
    yield r'score';
    yield serializers.serialize(
      object.score,
      specifiedType: const FullType(double),
    );
    yield r'thumbnail';
    yield serializers.serialize(
      object.thumbnail,
      specifiedType: const FullType(String),
    );
    yield r'asset';
    yield serializers.serialize(
      object.asset,
      specifiedType: const FullType(FaceDetailAsset),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    FaceDetail object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required FaceDetailBuilder result,
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
        case r'assetId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.assetId = valueDes;
          break;
        case r'score':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(double),
          ) as double;
          result.score = valueDes;
          break;
        case r'thumbnail':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.thumbnail = valueDes;
          break;
        case r'asset':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(FaceDetailAsset),
          ) as FaceDetailAsset;
          result.asset.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  FaceDetail deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = FaceDetailBuilder();
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



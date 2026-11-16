//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/face_bbox.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'face_box_patch_response.g.dart';

/// One committed face-box rewrite: the face's identity, its crop URL, and the stored box in the per-mille units the crop endpoint and the on-photo face-box layer address. 
///
/// Properties:
/// * [faceId] - Identifier of the rewritten face.
/// * [thumbnail] - URL of the face's crop, which now derives from the new box (the crop endpoint computes at query time; the change signal is the face row's updated_at). 
/// * [bbox] 
@BuiltValue()
abstract class FaceBoxPatchResponse implements Built<FaceBoxPatchResponse, FaceBoxPatchResponseBuilder> {
  /// Identifier of the rewritten face.
  @BuiltValueField(wireName: r'faceId')
  String get faceId;

  /// URL of the face's crop, which now derives from the new box (the crop endpoint computes at query time; the change signal is the face row's updated_at). 
  @BuiltValueField(wireName: r'thumbnail')
  String get thumbnail;

  @BuiltValueField(wireName: r'bbox')
  FaceBbox get bbox;

  FaceBoxPatchResponse._();

  factory FaceBoxPatchResponse([void updates(FaceBoxPatchResponseBuilder b)]) = _$FaceBoxPatchResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(FaceBoxPatchResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<FaceBoxPatchResponse> get serializer => _$FaceBoxPatchResponseSerializer();
}

class _$FaceBoxPatchResponseSerializer implements PrimitiveSerializer<FaceBoxPatchResponse> {
  @override
  final Iterable<Type> types = const [FaceBoxPatchResponse, _$FaceBoxPatchResponse];

  @override
  final String wireName = r'FaceBoxPatchResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    FaceBoxPatchResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'faceId';
    yield serializers.serialize(
      object.faceId,
      specifiedType: const FullType(String),
    );
    yield r'thumbnail';
    yield serializers.serialize(
      object.thumbnail,
      specifiedType: const FullType(String),
    );
    yield r'bbox';
    yield serializers.serialize(
      object.bbox,
      specifiedType: const FullType(FaceBbox),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    FaceBoxPatchResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required FaceBoxPatchResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'faceId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.faceId = valueDes;
          break;
        case r'thumbnail':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.thumbnail = valueDes;
          break;
        case r'bbox':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(FaceBbox),
          ) as FaceBbox;
          result.bbox.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  FaceBoxPatchResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = FaceBoxPatchResponseBuilder();
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



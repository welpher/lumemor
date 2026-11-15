//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/face_bbox.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'face_create_request.g.dart';

/// One hand-drawn face: the asset it sits on and the drawn box, in the FaceBbox per-mille units. The drawn box is the face row's bounding box as-is; the crop endpoint and the on-photo face-box layer render from it. 
///
/// Properties:
/// * [assetId] - The asset the drawn face sits on. It must be the caller's own live asset (the face writes' gate). 
/// * [bbox] 
@BuiltValue()
abstract class FaceCreateRequest implements Built<FaceCreateRequest, FaceCreateRequestBuilder> {
  /// The asset the drawn face sits on. It must be the caller's own live asset (the face writes' gate). 
  @BuiltValueField(wireName: r'assetId')
  String get assetId;

  @BuiltValueField(wireName: r'bbox')
  FaceBbox get bbox;

  FaceCreateRequest._();

  factory FaceCreateRequest([void updates(FaceCreateRequestBuilder b)]) = _$FaceCreateRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(FaceCreateRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<FaceCreateRequest> get serializer => _$FaceCreateRequestSerializer();
}

class _$FaceCreateRequestSerializer implements PrimitiveSerializer<FaceCreateRequest> {
  @override
  final Iterable<Type> types = const [FaceCreateRequest, _$FaceCreateRequest];

  @override
  final String wireName = r'FaceCreateRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    FaceCreateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'assetId';
    yield serializers.serialize(
      object.assetId,
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
    FaceCreateRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required FaceCreateRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'assetId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.assetId = valueDes;
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
  FaceCreateRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = FaceCreateRequestBuilder();
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



//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'cluster_face_response.g.dart';

/// One committed from-face person creation: the seeding face occupies the new cluster alone and serves as its representative face. 
///
/// Properties:
/// * [faceId] - Identifier of the seeding face.
/// * [clusterId] - The newly created cluster: the face is its sole member and its representative face. 
@BuiltValue()
abstract class ClusterFaceResponse implements Built<ClusterFaceResponse, ClusterFaceResponseBuilder> {
  /// Identifier of the seeding face.
  @BuiltValueField(wireName: r'faceId')
  String get faceId;

  /// The newly created cluster: the face is its sole member and its representative face. 
  @BuiltValueField(wireName: r'clusterId')
  String get clusterId;

  ClusterFaceResponse._();

  factory ClusterFaceResponse([void updates(ClusterFaceResponseBuilder b)]) = _$ClusterFaceResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ClusterFaceResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ClusterFaceResponse> get serializer => _$ClusterFaceResponseSerializer();
}

class _$ClusterFaceResponseSerializer implements PrimitiveSerializer<ClusterFaceResponse> {
  @override
  final Iterable<Type> types = const [ClusterFaceResponse, _$ClusterFaceResponse];

  @override
  final String wireName = r'ClusterFaceResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ClusterFaceResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'faceId';
    yield serializers.serialize(
      object.faceId,
      specifiedType: const FullType(String),
    );
    yield r'clusterId';
    yield serializers.serialize(
      object.clusterId,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ClusterFaceResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ClusterFaceResponseBuilder result,
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
        case r'clusterId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.clusterId = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ClusterFaceResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ClusterFaceResponseBuilder();
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



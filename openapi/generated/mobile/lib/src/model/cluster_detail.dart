//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:lumemor_api/src/model/person_label.dart';
import 'package:lumemor_api/src/model/face_detail.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'cluster_detail.g.dart';

/// One face cluster's detail as the calling user sees it: the caller's own label state and the cluster's faces on caller-visible assets, strongest score first. 
///
/// Properties:
/// * [clusterId] - Identifier of the face cluster.
/// * [label] - The caller's label of the cluster, or null when the caller has not labeled it (the unnamed state). It is the calling user's label or no label at all: another account's label never shows here (B4). 
/// * [faces] - The cluster's faces on assets visible to the caller, strongest score first with the face id as the tiebreak. Faces on other accounts' assets never appear. 
@BuiltValue()
abstract class ClusterDetail implements Built<ClusterDetail, ClusterDetailBuilder> {
  /// Identifier of the face cluster.
  @BuiltValueField(wireName: r'clusterId')
  String get clusterId;

  /// The caller's label of the cluster, or null when the caller has not labeled it (the unnamed state). It is the calling user's label or no label at all: another account's label never shows here (B4). 
  @BuiltValueField(wireName: r'label')
  PersonLabel? get label;

  /// The cluster's faces on assets visible to the caller, strongest score first with the face id as the tiebreak. Faces on other accounts' assets never appear. 
  @BuiltValueField(wireName: r'faces')
  BuiltList<FaceDetail> get faces;

  ClusterDetail._();

  factory ClusterDetail([void updates(ClusterDetailBuilder b)]) = _$ClusterDetail;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ClusterDetailBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ClusterDetail> get serializer => _$ClusterDetailSerializer();
}

class _$ClusterDetailSerializer implements PrimitiveSerializer<ClusterDetail> {
  @override
  final Iterable<Type> types = const [ClusterDetail, _$ClusterDetail];

  @override
  final String wireName = r'ClusterDetail';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ClusterDetail object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'clusterId';
    yield serializers.serialize(
      object.clusterId,
      specifiedType: const FullType(String),
    );
    yield r'label';
    yield object.label == null ? null : serializers.serialize(
      object.label,
      specifiedType: const FullType.nullable(PersonLabel),
    );
    yield r'faces';
    yield serializers.serialize(
      object.faces,
      specifiedType: const FullType(BuiltList, [FullType(FaceDetail)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ClusterDetail object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ClusterDetailBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'clusterId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.clusterId = valueDes;
          break;
        case r'label':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(PersonLabel),
          ) as PersonLabel?;
          if (valueDes == null) continue;
          result.label.replace(valueDes);
          break;
        case r'faces':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(FaceDetail)]),
          ) as BuiltList<FaceDetail>;
          result.faces.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ClusterDetail deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ClusterDetailBuilder();
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



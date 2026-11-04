//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'unnamed_cluster_summary.g.dart';

/// One unnamed face cluster of the caller's unnamed region: a global cluster the caller sees through at least one visible asset and has not labeled themselves. Unlabeled for THIS caller says nothing about another account's labels (B4 -- labels are private per user). 
///
/// Properties:
/// * [clusterId] - Identifier of the face cluster.
/// * [faceCount] - The cluster's faces on assets visible to the caller -- the honest per-caller count, not the global member count. 
/// * [representativeFaceId] - The cluster's representative face (its strongest-scoring member); the face the cluster tile renders. 
/// * [thumbnail] - URL of the representative face's crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with the representative face's id. 
@BuiltValue()
abstract class UnnamedClusterSummary implements Built<UnnamedClusterSummary, UnnamedClusterSummaryBuilder> {
  /// Identifier of the face cluster.
  @BuiltValueField(wireName: r'clusterId')
  String get clusterId;

  /// The cluster's faces on assets visible to the caller -- the honest per-caller count, not the global member count. 
  @BuiltValueField(wireName: r'faceCount')
  int get faceCount;

  /// The cluster's representative face (its strongest-scoring member); the face the cluster tile renders. 
  @BuiltValueField(wireName: r'representativeFaceId')
  String get representativeFaceId;

  /// URL of the representative face's crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with the representative face's id. 
  @BuiltValueField(wireName: r'thumbnail')
  String get thumbnail;

  UnnamedClusterSummary._();

  factory UnnamedClusterSummary([void updates(UnnamedClusterSummaryBuilder b)]) = _$UnnamedClusterSummary;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(UnnamedClusterSummaryBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<UnnamedClusterSummary> get serializer => _$UnnamedClusterSummarySerializer();
}

class _$UnnamedClusterSummarySerializer implements PrimitiveSerializer<UnnamedClusterSummary> {
  @override
  final Iterable<Type> types = const [UnnamedClusterSummary, _$UnnamedClusterSummary];

  @override
  final String wireName = r'UnnamedClusterSummary';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    UnnamedClusterSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'clusterId';
    yield serializers.serialize(
      object.clusterId,
      specifiedType: const FullType(String),
    );
    yield r'faceCount';
    yield serializers.serialize(
      object.faceCount,
      specifiedType: const FullType(int),
    );
    yield r'representativeFaceId';
    yield serializers.serialize(
      object.representativeFaceId,
      specifiedType: const FullType(String),
    );
    yield r'thumbnail';
    yield serializers.serialize(
      object.thumbnail,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    UnnamedClusterSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required UnnamedClusterSummaryBuilder result,
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
        case r'faceCount':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.faceCount = valueDes;
          break;
        case r'representativeFaceId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.representativeFaceId = valueDes;
          break;
        case r'thumbnail':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.thumbnail = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  UnnamedClusterSummary deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = UnnamedClusterSummaryBuilder();
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



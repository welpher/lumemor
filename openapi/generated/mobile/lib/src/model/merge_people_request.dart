//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'merge_people_request.g.dart';

/// One merge: 1-50 source clusters folded into one target cluster. The source clusters dissolve with their labels (the honest documented cost -- a source cluster's user-given name dies); the target's label persists. 
///
/// Properties:
/// * [sourceClusterIds] - The clusters to fold into the target: at least 1 and at most 50 after duplicates collapse, every one visible to the caller. Each dissolves with its label rows. A single id is the pairwise fold of the cluster detail's \"merge into\" action. 
/// * [targetClusterId] - The surviving cluster: every source face re-homes here and its representative recomputes to the strongest member of the merged set. It must be visible to the caller and must not sit among the sources. 
@BuiltValue()
abstract class MergePeopleRequest implements Built<MergePeopleRequest, MergePeopleRequestBuilder> {
  /// The clusters to fold into the target: at least 1 and at most 50 after duplicates collapse, every one visible to the caller. Each dissolves with its label rows. A single id is the pairwise fold of the cluster detail's \"merge into\" action. 
  @BuiltValueField(wireName: r'sourceClusterIds')
  BuiltList<String> get sourceClusterIds;

  /// The surviving cluster: every source face re-homes here and its representative recomputes to the strongest member of the merged set. It must be visible to the caller and must not sit among the sources. 
  @BuiltValueField(wireName: r'targetClusterId')
  String get targetClusterId;

  MergePeopleRequest._();

  factory MergePeopleRequest([void updates(MergePeopleRequestBuilder b)]) = _$MergePeopleRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(MergePeopleRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<MergePeopleRequest> get serializer => _$MergePeopleRequestSerializer();
}

class _$MergePeopleRequestSerializer implements PrimitiveSerializer<MergePeopleRequest> {
  @override
  final Iterable<Type> types = const [MergePeopleRequest, _$MergePeopleRequest];

  @override
  final String wireName = r'MergePeopleRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    MergePeopleRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'sourceClusterIds';
    yield serializers.serialize(
      object.sourceClusterIds,
      specifiedType: const FullType(BuiltList, [FullType(String)]),
    );
    yield r'targetClusterId';
    yield serializers.serialize(
      object.targetClusterId,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    MergePeopleRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required MergePeopleRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'sourceClusterIds':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(String)]),
          ) as BuiltList<String>;
          result.sourceClusterIds.replace(valueDes);
          break;
        case r'targetClusterId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.targetClusterId = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  MergePeopleRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = MergePeopleRequestBuilder();
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



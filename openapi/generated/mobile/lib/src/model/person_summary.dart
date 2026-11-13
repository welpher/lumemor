//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'person_summary.g.dart';

/// One named person of the caller's people list: a global face cluster carrying the caller's own label. The name is the caller's private label state -- another account labeling the same cluster never changes this view. The person entity's own attributes (the avatar) are shared: every account of the family sees the same avatar face. 
///
/// Properties:
/// * [clusterId] - Identifier of the face cluster.
/// * [name] - The caller's own label of the cluster.
/// * [photoCount] - The number of DISTINCT live assets the cluster has a face on that the caller can see -- photos, not faces, so two faces of one person in one photo count once. Honest per caller, never the global count. 
/// * [representativeFaceId] - The face the person card renders: the person's chosen avatar face when it is a caller-visible live member of the cluster, otherwise the cluster's strongest-scoring caller-visible face (score descending, the face id ascending as the tiebreak) -- the best face the CALLER can see, never a face on an asset the caller cannot view. The field is the thumbnail's face id. 
/// * [thumbnail] - URL of the card's face crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with `representativeFaceId`. 
@BuiltValue()
abstract class PersonSummary implements Built<PersonSummary, PersonSummaryBuilder> {
  /// Identifier of the face cluster.
  @BuiltValueField(wireName: r'clusterId')
  String get clusterId;

  /// The caller's own label of the cluster.
  @BuiltValueField(wireName: r'name')
  String get name;

  /// The number of DISTINCT live assets the cluster has a face on that the caller can see -- photos, not faces, so two faces of one person in one photo count once. Honest per caller, never the global count. 
  @BuiltValueField(wireName: r'photoCount')
  int get photoCount;

  /// The face the person card renders: the person's chosen avatar face when it is a caller-visible live member of the cluster, otherwise the cluster's strongest-scoring caller-visible face (score descending, the face id ascending as the tiebreak) -- the best face the CALLER can see, never a face on an asset the caller cannot view. The field is the thumbnail's face id. 
  @BuiltValueField(wireName: r'representativeFaceId')
  String get representativeFaceId;

  /// URL of the card's face crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with `representativeFaceId`. 
  @BuiltValueField(wireName: r'thumbnail')
  String get thumbnail;

  PersonSummary._();

  factory PersonSummary([void updates(PersonSummaryBuilder b)]) = _$PersonSummary;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(PersonSummaryBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<PersonSummary> get serializer => _$PersonSummarySerializer();
}

class _$PersonSummarySerializer implements PrimitiveSerializer<PersonSummary> {
  @override
  final Iterable<Type> types = const [PersonSummary, _$PersonSummary];

  @override
  final String wireName = r'PersonSummary';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    PersonSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'clusterId';
    yield serializers.serialize(
      object.clusterId,
      specifiedType: const FullType(String),
    );
    yield r'name';
    yield serializers.serialize(
      object.name,
      specifiedType: const FullType(String),
    );
    yield r'photoCount';
    yield serializers.serialize(
      object.photoCount,
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
    PersonSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required PersonSummaryBuilder result,
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
        case r'name':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.name = valueDes;
          break;
        case r'photoCount':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.photoCount = valueDes;
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
  PersonSummary deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = PersonSummaryBuilder();
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



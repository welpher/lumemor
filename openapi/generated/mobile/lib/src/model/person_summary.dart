//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'person_summary.g.dart';

/// One named person of the caller's people list: a global face cluster carrying the caller's own label. The name and the hidden switch are the caller's private label state (B4) -- another account labeling or hiding the same cluster never changes this view. 
///
/// Properties:
/// * [clusterId] - Identifier of the face cluster.
/// * [name] - The caller's own label of the cluster.
/// * [faceCount] - The cluster's faces on assets visible to the caller -- the honest per-caller count, not the global member count. 
/// * [representativeFaceId] - The cluster's representative face (its strongest-scoring member); the face the person card renders. 
/// * [thumbnail] - URL of the representative face's crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with the representative face's id. 
/// * [isHidden] - The caller's hidden switch for this person. Hidden people stay listed carrying this flag (badge semantics); they are not filtered out. 
@BuiltValue()
abstract class PersonSummary implements Built<PersonSummary, PersonSummaryBuilder> {
  /// Identifier of the face cluster.
  @BuiltValueField(wireName: r'clusterId')
  String get clusterId;

  /// The caller's own label of the cluster.
  @BuiltValueField(wireName: r'name')
  String get name;

  /// The cluster's faces on assets visible to the caller -- the honest per-caller count, not the global member count. 
  @BuiltValueField(wireName: r'faceCount')
  int get faceCount;

  /// The cluster's representative face (its strongest-scoring member); the face the person card renders. 
  @BuiltValueField(wireName: r'representativeFaceId')
  String get representativeFaceId;

  /// URL of the representative face's crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with the representative face's id. 
  @BuiltValueField(wireName: r'thumbnail')
  String get thumbnail;

  /// The caller's hidden switch for this person. Hidden people stay listed carrying this flag (badge semantics); they are not filtered out. 
  @BuiltValueField(wireName: r'isHidden')
  bool get isHidden;

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
    yield r'isHidden';
    yield serializers.serialize(
      object.isHidden,
      specifiedType: const FullType(bool),
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
        case r'isHidden':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.isHidden = valueDes;
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



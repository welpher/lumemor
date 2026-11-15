//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/date.dart';
import 'package:lumemor_api/src/model/person_label.dart';
import 'package:lumemor_api/src/model/face_bbox.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'asset_face_summary.g.dart';

/// One face of the asset detail's `faces` array: the crop's identity, the detection score, the face's bounding box (the on-photo face-box layer's geometry), the face's cluster, the CALLER's label of that cluster and the person entity's birth date. The label is per-user state: another account's label of the same global cluster never shows here; the birth date is the entity's own attribute (one value for every account) and exists so the chip can render the person's age WHEN THIS PHOTO WAS TAKEN. `clusterId` is null while the face waits for its first clustering assignment; such a face has no person yet and every assignment write on it answers the people surface's 404. 
///
/// Properties:
/// * [id] - Identifier of the face row.
/// * [score] - The detector's confidence for this face; a hand-drawn face (the caller's own certainty) carries 1.0. 
/// * [thumbnail] - URL of the face's crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with this face's id, serving the bounding-box crop of the asset's view-size derivative. 
/// * [bbox] 
/// * [clusterId] - Identifier of the face's cluster, or null before the clustering assigned one (a face the recognition run has not grouped yet). 
/// * [label] - The caller's own label of the face's cluster, or null when the caller has not labeled it (the unnamed state) -- and null for a clusterless face, which has no cluster to label. 
/// * [birthDate] - The person's birth date (the entity's own attribute), or null when unset or the face has no cluster. The client computes the capture-time age from it and the asset's `takenAt` -- the birth date's whole purpose beside the birthday memories. 
@BuiltValue()
abstract class AssetFaceSummary implements Built<AssetFaceSummary, AssetFaceSummaryBuilder> {
  /// Identifier of the face row.
  @BuiltValueField(wireName: r'id')
  String get id;

  /// The detector's confidence for this face; a hand-drawn face (the caller's own certainty) carries 1.0. 
  @BuiltValueField(wireName: r'score')
  double get score;

  /// URL of the face's crop: the face crop endpoint `GET /api/v1/faces/{faceId}/thumbnail` with this face's id, serving the bounding-box crop of the asset's view-size derivative. 
  @BuiltValueField(wireName: r'thumbnail')
  String get thumbnail;

  @BuiltValueField(wireName: r'bbox')
  FaceBbox get bbox;

  /// Identifier of the face's cluster, or null before the clustering assigned one (a face the recognition run has not grouped yet). 
  @BuiltValueField(wireName: r'clusterId')
  String? get clusterId;

  /// The caller's own label of the face's cluster, or null when the caller has not labeled it (the unnamed state) -- and null for a clusterless face, which has no cluster to label. 
  @BuiltValueField(wireName: r'label')
  PersonLabel? get label;

  /// The person's birth date (the entity's own attribute), or null when unset or the face has no cluster. The client computes the capture-time age from it and the asset's `takenAt` -- the birth date's whole purpose beside the birthday memories. 
  @BuiltValueField(wireName: r'birthDate')
  Date? get birthDate;

  AssetFaceSummary._();

  factory AssetFaceSummary([void updates(AssetFaceSummaryBuilder b)]) = _$AssetFaceSummary;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AssetFaceSummaryBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AssetFaceSummary> get serializer => _$AssetFaceSummarySerializer();
}

class _$AssetFaceSummarySerializer implements PrimitiveSerializer<AssetFaceSummary> {
  @override
  final Iterable<Type> types = const [AssetFaceSummary, _$AssetFaceSummary];

  @override
  final String wireName = r'AssetFaceSummary';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AssetFaceSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
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
    yield r'bbox';
    yield serializers.serialize(
      object.bbox,
      specifiedType: const FullType(FaceBbox),
    );
    yield r'clusterId';
    yield object.clusterId == null ? null : serializers.serialize(
      object.clusterId,
      specifiedType: const FullType.nullable(String),
    );
    yield r'label';
    yield object.label == null ? null : serializers.serialize(
      object.label,
      specifiedType: const FullType.nullable(PersonLabel),
    );
    yield r'birthDate';
    yield object.birthDate == null ? null : serializers.serialize(
      object.birthDate,
      specifiedType: const FullType.nullable(Date),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    AssetFaceSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AssetFaceSummaryBuilder result,
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
        case r'bbox':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(FaceBbox),
          ) as FaceBbox;
          result.bbox.replace(valueDes);
          break;
        case r'clusterId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
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
        case r'birthDate':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(Date),
          ) as Date?;
          if (valueDes == null) continue;
          result.birthDate = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  AssetFaceSummary deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AssetFaceSummaryBuilder();
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



//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'merge_people_response.g.dart';

/// One committed merge.
///
/// Properties:
/// * [survivingClusterId] - The target cluster that absorbed the sources.
/// * [mergedClusterIds] - The source clusters that dissolved into the target, in request order. Their rows -- and every label on them -- are gone. 
@BuiltValue()
abstract class MergePeopleResponse implements Built<MergePeopleResponse, MergePeopleResponseBuilder> {
  /// The target cluster that absorbed the sources.
  @BuiltValueField(wireName: r'survivingClusterId')
  String get survivingClusterId;

  /// The source clusters that dissolved into the target, in request order. Their rows -- and every label on them -- are gone. 
  @BuiltValueField(wireName: r'mergedClusterIds')
  BuiltList<String> get mergedClusterIds;

  MergePeopleResponse._();

  factory MergePeopleResponse([void updates(MergePeopleResponseBuilder b)]) = _$MergePeopleResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(MergePeopleResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<MergePeopleResponse> get serializer => _$MergePeopleResponseSerializer();
}

class _$MergePeopleResponseSerializer implements PrimitiveSerializer<MergePeopleResponse> {
  @override
  final Iterable<Type> types = const [MergePeopleResponse, _$MergePeopleResponse];

  @override
  final String wireName = r'MergePeopleResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    MergePeopleResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'survivingClusterId';
    yield serializers.serialize(
      object.survivingClusterId,
      specifiedType: const FullType(String),
    );
    yield r'mergedClusterIds';
    yield serializers.serialize(
      object.mergedClusterIds,
      specifiedType: const FullType(BuiltList, [FullType(String)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    MergePeopleResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required MergePeopleResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'survivingClusterId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.survivingClusterId = valueDes;
          break;
        case r'mergedClusterIds':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(String)]),
          ) as BuiltList<String>;
          result.mergedClusterIds.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  MergePeopleResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = MergePeopleResponseBuilder();
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



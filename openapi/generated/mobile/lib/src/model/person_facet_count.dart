//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'person_facet_count.g.dart';

/// One named person of the people facet.
///
/// Properties:
/// * [id] - The face cluster id (the people surface's id).
/// * [name] - The CALLER's label of the cluster.
/// * [assetCount] - The number of assets of the current filtered set (minus the person filter) carrying a face of this cluster. 
@BuiltValue()
abstract class PersonFacetCount implements Built<PersonFacetCount, PersonFacetCountBuilder> {
  /// The face cluster id (the people surface's id).
  @BuiltValueField(wireName: r'id')
  String get id;

  /// The CALLER's label of the cluster.
  @BuiltValueField(wireName: r'name')
  String get name;

  /// The number of assets of the current filtered set (minus the person filter) carrying a face of this cluster. 
  @BuiltValueField(wireName: r'assetCount')
  int get assetCount;

  PersonFacetCount._();

  factory PersonFacetCount([void updates(PersonFacetCountBuilder b)]) = _$PersonFacetCount;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(PersonFacetCountBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<PersonFacetCount> get serializer => _$PersonFacetCountSerializer();
}

class _$PersonFacetCountSerializer implements PrimitiveSerializer<PersonFacetCount> {
  @override
  final Iterable<Type> types = const [PersonFacetCount, _$PersonFacetCount];

  @override
  final String wireName = r'PersonFacetCount';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    PersonFacetCount object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'name';
    yield serializers.serialize(
      object.name,
      specifiedType: const FullType(String),
    );
    yield r'assetCount';
    yield serializers.serialize(
      object.assetCount,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    PersonFacetCount object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required PersonFacetCountBuilder result,
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
        case r'name':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.name = valueDes;
          break;
        case r'assetCount':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.assetCount = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  PersonFacetCount deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = PersonFacetCountBuilder();
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



//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/media_kind.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'kind_facet_count.g.dart';

/// One media family of the kinds facet.
///
/// Properties:
/// * [kind] 
/// * [assetCount] - The number of assets of the current filtered set (minus the kind filter) of this family. 
@BuiltValue()
abstract class KindFacetCount implements Built<KindFacetCount, KindFacetCountBuilder> {
  @BuiltValueField(wireName: r'kind')
  MediaKind get kind;
  // enum kindEnum {  image,  video,  };

  /// The number of assets of the current filtered set (minus the kind filter) of this family. 
  @BuiltValueField(wireName: r'assetCount')
  int get assetCount;

  KindFacetCount._();

  factory KindFacetCount([void updates(KindFacetCountBuilder b)]) = _$KindFacetCount;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(KindFacetCountBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<KindFacetCount> get serializer => _$KindFacetCountSerializer();
}

class _$KindFacetCountSerializer implements PrimitiveSerializer<KindFacetCount> {
  @override
  final Iterable<Type> types = const [KindFacetCount, _$KindFacetCount];

  @override
  final String wireName = r'KindFacetCount';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    KindFacetCount object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'kind';
    yield serializers.serialize(
      object.kind,
      specifiedType: const FullType(MediaKind),
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
    KindFacetCount object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required KindFacetCountBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'kind':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(MediaKind),
          ) as MediaKind;
          result.kind = valueDes;
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
  KindFacetCount deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = KindFacetCountBuilder();
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



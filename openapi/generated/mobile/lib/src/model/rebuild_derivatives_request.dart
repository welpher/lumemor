//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'rebuild_derivatives_request.g.dart';

/// The scope of one derivative rebuild run: the honest pair the image card's dialog chooses between. 
///
/// Properties:
/// * [scope] - `missing` derives only the assets whose current-format serving files are absent; `all` re-derives every image asset at the current settings and removes the other encoding format's stale files. 
@BuiltValue()
abstract class RebuildDerivativesRequest implements Built<RebuildDerivativesRequest, RebuildDerivativesRequestBuilder> {
  /// `missing` derives only the assets whose current-format serving files are absent; `all` re-derives every image asset at the current settings and removes the other encoding format's stale files. 
  @BuiltValueField(wireName: r'scope')
  RebuildDerivativesRequestScopeEnum get scope;
  // enum scopeEnum {  missing,  all,  };

  RebuildDerivativesRequest._();

  factory RebuildDerivativesRequest([void updates(RebuildDerivativesRequestBuilder b)]) = _$RebuildDerivativesRequest;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(RebuildDerivativesRequestBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<RebuildDerivativesRequest> get serializer => _$RebuildDerivativesRequestSerializer();
}

class _$RebuildDerivativesRequestSerializer implements PrimitiveSerializer<RebuildDerivativesRequest> {
  @override
  final Iterable<Type> types = const [RebuildDerivativesRequest, _$RebuildDerivativesRequest];

  @override
  final String wireName = r'RebuildDerivativesRequest';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    RebuildDerivativesRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'scope';
    yield serializers.serialize(
      object.scope,
      specifiedType: const FullType(RebuildDerivativesRequestScopeEnum),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    RebuildDerivativesRequest object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required RebuildDerivativesRequestBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'scope':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(RebuildDerivativesRequestScopeEnum),
          ) as RebuildDerivativesRequestScopeEnum;
          result.scope = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  RebuildDerivativesRequest deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = RebuildDerivativesRequestBuilder();
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


/// `missing` derives only the assets whose current-format serving files are absent; `all` re-derives every image asset at the current settings and removes the other encoding format's stale files. 
class RebuildDerivativesRequestScopeEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'missing')
  static const RebuildDerivativesRequestScopeEnum missing = _$rebuildDerivativesRequestScopeEnum_missing;
  @BuiltValueEnumConst(wireName: r'all')
  static const RebuildDerivativesRequestScopeEnum all = _$rebuildDerivativesRequestScopeEnum_all;

  static Serializer<RebuildDerivativesRequestScopeEnum> get serializer => _$rebuildDerivativesRequestScopeEnumSerializer;

  const RebuildDerivativesRequestScopeEnum._(String name): super(name);

  static BuiltSet<RebuildDerivativesRequestScopeEnum> get values => _$rebuildDerivativesRequestScopeEnumValues;
  static RebuildDerivativesRequestScopeEnum valueOf(String name) => _$rebuildDerivativesRequestScopeEnumValueOf(name);
}


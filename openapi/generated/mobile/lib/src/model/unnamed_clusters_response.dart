//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/unnamed_cluster_summary.dart';
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'unnamed_clusters_response.g.dart';

/// One keyset page of the caller's unnamed clusters.
///
/// Properties:
/// * [items] - The page's clusters, ordered by `faceCount` descending with the cluster id as the tiebreak. 
/// * [nextCursor] - The opaque continuation cursor: pass it back as the `cursor` query parameter to fetch the next page. Absent exactly when this is the last page. 
/// * [totalCount] - The caller's whole unnamed region: every cluster this listing would walk, independent of the page size and the cursor. The people list's unnamed entry badge and the unnamed page's own count both read it. 
@BuiltValue()
abstract class UnnamedClustersResponse implements Built<UnnamedClustersResponse, UnnamedClustersResponseBuilder> {
  /// The page's clusters, ordered by `faceCount` descending with the cluster id as the tiebreak. 
  @BuiltValueField(wireName: r'items')
  BuiltList<UnnamedClusterSummary> get items;

  /// The opaque continuation cursor: pass it back as the `cursor` query parameter to fetch the next page. Absent exactly when this is the last page. 
  @BuiltValueField(wireName: r'nextCursor')
  String? get nextCursor;

  /// The caller's whole unnamed region: every cluster this listing would walk, independent of the page size and the cursor. The people list's unnamed entry badge and the unnamed page's own count both read it. 
  @BuiltValueField(wireName: r'totalCount')
  int get totalCount;

  UnnamedClustersResponse._();

  factory UnnamedClustersResponse([void updates(UnnamedClustersResponseBuilder b)]) = _$UnnamedClustersResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(UnnamedClustersResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<UnnamedClustersResponse> get serializer => _$UnnamedClustersResponseSerializer();
}

class _$UnnamedClustersResponseSerializer implements PrimitiveSerializer<UnnamedClustersResponse> {
  @override
  final Iterable<Type> types = const [UnnamedClustersResponse, _$UnnamedClustersResponse];

  @override
  final String wireName = r'UnnamedClustersResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    UnnamedClustersResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'items';
    yield serializers.serialize(
      object.items,
      specifiedType: const FullType(BuiltList, [FullType(UnnamedClusterSummary)]),
    );
    if (object.nextCursor != null) {
      yield r'nextCursor';
      yield serializers.serialize(
        object.nextCursor,
        specifiedType: const FullType(String),
      );
    }
    yield r'totalCount';
    yield serializers.serialize(
      object.totalCount,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    UnnamedClustersResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required UnnamedClustersResponseBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'items':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BuiltList, [FullType(UnnamedClusterSummary)]),
          ) as BuiltList<UnnamedClusterSummary>;
          result.items.replace(valueDes);
          break;
        case r'nextCursor':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.nextCursor = valueDes;
          break;
        case r'totalCount':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.totalCount = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  UnnamedClustersResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = UnnamedClustersResponseBuilder();
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



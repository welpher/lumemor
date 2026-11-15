//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/download_format.dart';
import 'package:lumemor_api/src/model/user_locale.dart';
import 'package:lumemor_api/src/model/user_theme.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'user_self_patch.g.dart';

/// Payload of `PATCH /users/me`. Every field is optional: an absent field keeps its stored value, a present one replaces it. At least one field must be present. 
///
/// Properties:
/// * [name] - The new display name; 1 to 100 runes after trimming (an empty name is refused -- the profile always carries one). 
/// * [email] - The new login address, parsed and lower-cased like every creation path; an address another account already owns answers 409 USER_EMAIL_TAKEN. 
/// * [theme] 
/// * [locale] 
/// * [downloadFormat] 
/// * [downloadIncludeVideos] - The download packaging's video switch.
/// * [downloadIncludeSidecars] - The download packaging's sidecar switch.
/// * [downloadIncludeEmbeddedVideos] - The download packaging's embedded-motion switch (default false: the single-file motion photo's original already carries its motion). 
@BuiltValue()
abstract class UserSelfPatch implements Built<UserSelfPatch, UserSelfPatchBuilder> {
  /// The new display name; 1 to 100 runes after trimming (an empty name is refused -- the profile always carries one). 
  @BuiltValueField(wireName: r'name')
  String? get name;

  /// The new login address, parsed and lower-cased like every creation path; an address another account already owns answers 409 USER_EMAIL_TAKEN. 
  @BuiltValueField(wireName: r'email')
  String? get email;

  @BuiltValueField(wireName: r'theme')
  UserTheme? get theme;
  // enum themeEnum {  auto,  light,  dark,  };

  @BuiltValueField(wireName: r'locale')
  UserLocale? get locale;
  // enum localeEnum {  en,  zh-Hans,  };

  @BuiltValueField(wireName: r'downloadFormat')
  DownloadFormat? get downloadFormat;
  // enum downloadFormatEnum {  zip,  tar,  };

  /// The download packaging's video switch.
  @BuiltValueField(wireName: r'downloadIncludeVideos')
  bool? get downloadIncludeVideos;

  /// The download packaging's sidecar switch.
  @BuiltValueField(wireName: r'downloadIncludeSidecars')
  bool? get downloadIncludeSidecars;

  /// The download packaging's embedded-motion switch (default false: the single-file motion photo's original already carries its motion). 
  @BuiltValueField(wireName: r'downloadIncludeEmbeddedVideos')
  bool? get downloadIncludeEmbeddedVideos;

  UserSelfPatch._();

  factory UserSelfPatch([void updates(UserSelfPatchBuilder b)]) = _$UserSelfPatch;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(UserSelfPatchBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<UserSelfPatch> get serializer => _$UserSelfPatchSerializer();
}

class _$UserSelfPatchSerializer implements PrimitiveSerializer<UserSelfPatch> {
  @override
  final Iterable<Type> types = const [UserSelfPatch, _$UserSelfPatch];

  @override
  final String wireName = r'UserSelfPatch';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    UserSelfPatch object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    if (object.name != null) {
      yield r'name';
      yield serializers.serialize(
        object.name,
        specifiedType: const FullType(String),
      );
    }
    if (object.email != null) {
      yield r'email';
      yield serializers.serialize(
        object.email,
        specifiedType: const FullType(String),
      );
    }
    if (object.theme != null) {
      yield r'theme';
      yield serializers.serialize(
        object.theme,
        specifiedType: const FullType(UserTheme),
      );
    }
    if (object.locale != null) {
      yield r'locale';
      yield serializers.serialize(
        object.locale,
        specifiedType: const FullType(UserLocale),
      );
    }
    if (object.downloadFormat != null) {
      yield r'downloadFormat';
      yield serializers.serialize(
        object.downloadFormat,
        specifiedType: const FullType(DownloadFormat),
      );
    }
    if (object.downloadIncludeVideos != null) {
      yield r'downloadIncludeVideos';
      yield serializers.serialize(
        object.downloadIncludeVideos,
        specifiedType: const FullType(bool),
      );
    }
    if (object.downloadIncludeSidecars != null) {
      yield r'downloadIncludeSidecars';
      yield serializers.serialize(
        object.downloadIncludeSidecars,
        specifiedType: const FullType(bool),
      );
    }
    if (object.downloadIncludeEmbeddedVideos != null) {
      yield r'downloadIncludeEmbeddedVideos';
      yield serializers.serialize(
        object.downloadIncludeEmbeddedVideos,
        specifiedType: const FullType(bool),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    UserSelfPatch object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required UserSelfPatchBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'name':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.name = valueDes;
          break;
        case r'email':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.email = valueDes;
          break;
        case r'theme':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(UserTheme),
          ) as UserTheme?;
          if (valueDes == null) continue;
          result.theme = valueDes;
          break;
        case r'locale':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(UserLocale),
          ) as UserLocale?;
          if (valueDes == null) continue;
          result.locale = valueDes;
          break;
        case r'downloadFormat':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(DownloadFormat),
          ) as DownloadFormat?;
          if (valueDes == null) continue;
          result.downloadFormat = valueDes;
          break;
        case r'downloadIncludeVideos':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(bool),
          ) as bool?;
          if (valueDes == null) continue;
          result.downloadIncludeVideos = valueDes;
          break;
        case r'downloadIncludeSidecars':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(bool),
          ) as bool?;
          if (valueDes == null) continue;
          result.downloadIncludeSidecars = valueDes;
          break;
        case r'downloadIncludeEmbeddedVideos':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(bool),
          ) as bool?;
          if (valueDes == null) continue;
          result.downloadIncludeEmbeddedVideos = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  UserSelfPatch deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = UserSelfPatchBuilder();
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



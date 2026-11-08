//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:lumemor_api/src/model/user_role.dart';
import 'package:lumemor_api/src/model/download_format.dart';
import 'package:lumemor_api/src/model/user_locale.dart';
import 'package:lumemor_api/src/model/user_theme.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'user_self_response.g.dart';

/// The caller's own account as the me-scoped user surface returns it: the identity fields of the profile section, the appearance pair, and the download preferences. The password hash is never part of any response. 
///
/// Properties:
/// * [id] - UUIDv7 identifier of the account.
/// * [name] - Display name.
/// * [email] - Login address; unique instance-wide.
/// * [role] 
/// * [theme] 
/// * [locale] 
/// * [downloadFormat] 
/// * [downloadIncludeVideos] - Whether the account's download packaging includes the original video files. Photos are the download's content and always ride along. 
/// * [downloadIncludeSidecars] - Whether the account's download packaging includes EXIF sidecar files (`.xmp`) beside the media. 
@BuiltValue()
abstract class UserSelfResponse implements Built<UserSelfResponse, UserSelfResponseBuilder> {
  /// UUIDv7 identifier of the account.
  @BuiltValueField(wireName: r'id')
  String get id;

  /// Display name.
  @BuiltValueField(wireName: r'name')
  String get name;

  /// Login address; unique instance-wide.
  @BuiltValueField(wireName: r'email')
  String get email;

  @BuiltValueField(wireName: r'role')
  UserRole get role;
  // enum roleEnum {  admin,  user,  };

  @BuiltValueField(wireName: r'theme')
  UserTheme get theme;
  // enum themeEnum {  auto,  light,  dark,  };

  @BuiltValueField(wireName: r'locale')
  UserLocale get locale;
  // enum localeEnum {  en,  zh-Hans,  };

  @BuiltValueField(wireName: r'downloadFormat')
  DownloadFormat get downloadFormat;
  // enum downloadFormatEnum {  zip,  tar,  };

  /// Whether the account's download packaging includes the original video files. Photos are the download's content and always ride along. 
  @BuiltValueField(wireName: r'downloadIncludeVideos')
  bool get downloadIncludeVideos;

  /// Whether the account's download packaging includes EXIF sidecar files (`.xmp`) beside the media. 
  @BuiltValueField(wireName: r'downloadIncludeSidecars')
  bool get downloadIncludeSidecars;

  UserSelfResponse._();

  factory UserSelfResponse([void updates(UserSelfResponseBuilder b)]) = _$UserSelfResponse;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(UserSelfResponseBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<UserSelfResponse> get serializer => _$UserSelfResponseSerializer();
}

class _$UserSelfResponseSerializer implements PrimitiveSerializer<UserSelfResponse> {
  @override
  final Iterable<Type> types = const [UserSelfResponse, _$UserSelfResponse];

  @override
  final String wireName = r'UserSelfResponse';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    UserSelfResponse object, {
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
    yield r'email';
    yield serializers.serialize(
      object.email,
      specifiedType: const FullType(String),
    );
    yield r'role';
    yield serializers.serialize(
      object.role,
      specifiedType: const FullType(UserRole),
    );
    yield r'theme';
    yield serializers.serialize(
      object.theme,
      specifiedType: const FullType(UserTheme),
    );
    yield r'locale';
    yield serializers.serialize(
      object.locale,
      specifiedType: const FullType(UserLocale),
    );
    yield r'downloadFormat';
    yield serializers.serialize(
      object.downloadFormat,
      specifiedType: const FullType(DownloadFormat),
    );
    yield r'downloadIncludeVideos';
    yield serializers.serialize(
      object.downloadIncludeVideos,
      specifiedType: const FullType(bool),
    );
    yield r'downloadIncludeSidecars';
    yield serializers.serialize(
      object.downloadIncludeSidecars,
      specifiedType: const FullType(bool),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    UserSelfResponse object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required UserSelfResponseBuilder result,
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
        case r'email':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.email = valueDes;
          break;
        case r'role':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(UserRole),
          ) as UserRole;
          result.role = valueDes;
          break;
        case r'theme':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(UserTheme),
          ) as UserTheme;
          result.theme = valueDes;
          break;
        case r'locale':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(UserLocale),
          ) as UserLocale;
          result.locale = valueDes;
          break;
        case r'downloadFormat':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DownloadFormat),
          ) as DownloadFormat;
          result.downloadFormat = valueDes;
          break;
        case r'downloadIncludeVideos':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.downloadIncludeVideos = valueDes;
          break;
        case r'downloadIncludeSidecars':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.downloadIncludeSidecars = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  UserSelfResponse deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = UserSelfResponseBuilder();
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



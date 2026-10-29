import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';


/// tests for AssetsApi
void main() {
  final instance = LumemorApi().getAssetsApi();

  group(AssetsApi, () {
    // Look up an uploaded asset by its device identifier.
    //
    // The backup idempotency probe: the app asks, before uploading a local file, whether its `deviceAssetId` already exists for this account, and skips the upload when it does.  The lookup is owner-scoped -- a `deviceAssetId` of another user's asset is invisible and answers 404. The `state` field mirrors what a re-upload of the same content would produce: `duplicate` when the asset is live, `trashed` when it sits in the trash (the client can then offer restore or save-as-new before spending bandwidth). 404 means the device has not uploaded this identifier (or only under another account). 
    //
    //Future<AssetResponse> getAssetByDeviceId(String deviceAssetId) async
    test('test getAssetByDeviceId', () async {
      // TODO
    });

    // Upload one photo, optionally with its Live Photo video.
    //
    // The backup upload endpoint: one photo per request, streamed into content-addressed storage while it is read, with the Live Photo pairing video riding the same request as a second file part.  The request body is `multipart/form-data` with these parts:  - `deviceAssetId` (text, required) -- the stable identity the   uploading device assigned to the file; together with the   account it scopes the device idempotency of backups. - `fileName` (text, required) -- the original client-side file   name of the photo, recorded as the asset's   `originalFilename`. - `deviceCreatedAt` (text, optional) -- the capture time the   device reports, RFC 3339 with offset. It seeds the asset's   `takenAt` until the metadata pipeline refines it from EXIF. - `deviceTimeZone` (text, optional) -- the IANA time zone name   of the capture location; it feeds the metadata pipeline's   timezone fallback chain and is not validated at upload. - `photo` (file, required) -- the photo bytes. - `liveVideo` (file, optional) -- the pairing video of a Live   Photo; present only together with `photo`.  Part order is part of the contract: all text fields precede every file part, and the `liveVideo` part follows the `photo` part. The server streams file parts to disk without buffering them, so every field must be known and validated before the first file byte is accepted -- a file part arriving before the required fields, a text field after a file part, or a `liveVideo` without a preceding `photo` answers 400 `VALIDATION_FAILED`. Each text field value is capped at 1024 bytes; `deviceAssetId`, `fileName` and `deviceTimeZone` carry the length bounds below. The storage extension and the media kind derive from the *file part's own* file name, and the payload must be consistent with it: a renamed file of another type answers 400 `VALIDATION_FAILED`.  Uploads are idempotent through the owner-scoped checksum: re-uploading identical bytes never creates a second asset. The outcome is one of three states, discriminated by the `state` field of the response body:  - 201 `created`: no asset with this checksum existed; a new   asset was created (a Live Photo created two rows, the   response describes the photo). - 200 `duplicate`: an identical asset of the same owner is   live; that asset is returned. - 200 `trashed`: an identical asset of the same owner sits in   the trash; that asset is returned (with `isTrashed: true`)   so the client can offer the user restore or save-as-new.  Each file is capped at the instance's per-file upload limit (2 GiB by default); a file over the limit aborts mid-stream with 413 `ASSET_TOO_LARGE`. This endpoint takes multipart bodies, not JSON, and is deliberately exempt from the API-wide JSON body cap; the per-file upload limit governs instead. 
    //
    //Future<AssetResponse> uploadAsset(String deviceAssetId, String fileName, MultipartFile photo, { DateTime deviceCreatedAt, String deviceTimeZone, MultipartFile liveVideo }) async
    test('test uploadAsset', () async {
      // TODO
    });

  });
}

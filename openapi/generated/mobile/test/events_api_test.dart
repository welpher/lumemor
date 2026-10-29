import 'package:test/test.dart';
import 'package:lumemor_api/lumemor_api.dart';


/// tests for EventsApi
void main() {
  final instance = LumemorApi().getEventsApi();

  group(EventsApi, () {
    // Stream instance activity over server-sent events.
    //
    // The activity stream of the instance, authenticated like the REST API: the web session cookie and the mobile bearer token both open the stream.  Every event is one `text/event-stream` block: `event:` carries the event name, `id:` a monotonically increasing number, and `data:` the JSON payload -- the resource id plus a summary of the changed fields. Event names follow `<resource>.<past-tense verb>`:  - `asset.uploaded`, `asset.ready`, `asset.updated`,   `asset.trashed`, `asset.restored`, `asset.deleted` - `album.created`, `album.updated`, `album.deleted`,   `album.items-changed`, `album.shared`, `album.unshared` - `face.cluster-updated` - `job.progress`, `job.finished` - `model.changed` - `session.started`, `session.revoked` - `server.settings-changed`  The server keeps the last five minutes of events in a ring buffer. Browser EventSource clients reconnect automatically and resend `Last-Event-ID`; the server replays everything newer than that id, and events that fell out of the ring are skipped. 
    //
    //Future<String> getEvents({ String lastEventID }) async
    test('test getEvents', () async {
      // TODO
    });

  });
}

# mobile

Flutter app.

- Riverpod for state management; drift for the offline timeline cache
- photomanager for library access; background_downloader for background backup (Android WorkManager / iOS BGTaskScheduler)
- First-run pairing: mDNS discovery + QR code + manual URL fallback
- API clients are generated from the root `openapi/` spec (dart-dio) — never hand-written

# jobs testdata provenance

| File | Origin | License / terms |
| --- | --- | --- |
| `golden.jpg` | Self-made fixture: a minimal JPEG whose EXIF carries the camera identity (Make "Lumemor", Model "Golden Camera", the lens "Golden 35mm f/2.8"), the capture wall clock `2023:07:14 10:15:30` with `OffsetTimeOriginal +05:30`, and GPS coordinates at the Eiffel Tower (48°51'30.24"N, 2°17'40.2"E -- resolving to GeoNames 2988394, Paris 16 Passy, under the place bake). | None (self-made content, no third-party material). |
| `golden-nooffset.jpg`, `golden-offsetoriginal.jpg` | Derivatives of `golden.jpg`: the same frame without the offset tags / with only OffsetTimeOriginal, the time-resolution chain's fixture pair. | None (as above). |
| `gps-beijing.jpg` | Derivative of `golden.jpg`: the same EXIF with the GPS rationals rewritten to Tiananmen Square (39°54'19.44"N, 116°23'51.36"E -- resolving to GeoNames 1816670, Beijing, the zh-Hans-named city) and DateTimeOriginal rewritten to `2026:06:20 10:15:30` (a byte-identical-length in-place patch, so every other field is untouched). The acceptance chain's GPS-bearing photo: the place resolution, the place filter and the places page's city. | None (as above). |

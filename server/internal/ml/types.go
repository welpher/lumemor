// The shared value type of the inference wrappers (the four
// inference jobs: face detection, face recognition, CLIP embedding and OCR all
// consume decoded images).
package ml

// Image is a decoded RGB raster: 8 bits per channel, three channels,
// row-major, no padding. Width and Height are in pixels, so
// len(Pixels) == Width*Height*3. The decode from the stored original
// (including the EXIF orientation fix) lives with the job that owns
// the decoding; the wrappers and their tests work on the decoded
// raster only.
type Image struct {
	Width  int
	Height int
	// Pixels holds the RGB data, row-major.
	Pixels []byte
}

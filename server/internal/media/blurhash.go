package media

// The blurhash parameters of the serving pipeline: the hash is a
// placeholder gradient, 4x3 components (the reference default), computed
// from the tile-sized image -- small enough to encode fast, close
// enough to the rendered tile to placeholder it.
const (
	blurhashXComponents = 4
	blurhashYComponents = 3
)

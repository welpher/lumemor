package jobs

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// THE CONTENT IDENTIFIER PAIRING: the second entrance onto the same
// pair. An iPhone (and some Android cameras) stamps both halves of a
// Live Photo with one identifier -- ContentIdentifier, or
// MediaGroupUUID on the Android side -- so two files that arrive as
// two separate uploads (a browser drop, an interrupted backup, a
// migration) can still find each other. The reference is
// (owner, identifier, kind): the same identifier never matches across
// accounts, and a name-only resemblance never matches at all.
//
// The match runs at the END of the metadata walk, after the row's own
// identifier landed. A miss is the ordinary case -- the other half has
// simply not arrived yet -- and stays silent, waiting for the other
// half's own metadata run to find THIS row.

// maxLivePhotoCIDRunes bounds the identifier the way every other
// client-supplied string is bounded: a file tag is still a file's
// bytes, and the column feeds an index.
const maxLivePhotoCIDRunes = 512

// livePhotoCID reads the pairing identifier from one extraction: the
// iOS tag first, the Android one second. Empty (no tag, blank, or
// absurdly long) answers no identifier and the row pairs by nothing.
func livePhotoCID(tags map[string]any) string {
	for _, name := range []string{"ContentIdentifier", "MediaGroupUUID"} {
		value, ok := tagString(tags, name)
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || utf8.RuneCountInString(trimmed) > maxLivePhotoCIDRunes {
			continue
		}
		return trimmed
	}
	return ""
}

// pairLivePhotoByCID hunts the other half of one asset's pair and
// links the two when both are present. It runs after the metadata
// transaction committed (the row's identifier must be readable, and
// the counterpart may be any earlier row), and it never fails the
// job: a pair that did not form is not an error, it is a wait.
func (w *MetadataWorker) pairLivePhotoByCID(ctx context.Context, assetID uuid.UUID, cid string) {
	queries := dbgen.New(w.pool)

	self, err := queries.GetAssetForMotionPhoto(ctx, pgUUID(assetID))
	if errors.Is(err, pgx.ErrNoRows) {
		return
	}
	if err != nil {
		w.logger.Error("read the asset for the Live Photo pairing", "asset_id", assetID, "error", err)
		return
	}
	// A component row is already somebody's motion: the extraction
	// built it, or an earlier pairing did. It never pairs again.
	if self.IsComponent {
		return
	}

	otherKind := dbgen.MediaKindVideo
	if self.MediaKind == dbgen.MediaKindVideo {
		otherKind = dbgen.MediaKindImage
	}
	match, err := queries.GetLivePhotoCIDMatch(ctx, dbgen.GetLivePhotoCIDMatchParams{
		OwnerID:      self.OwnerID,
		LivePhotoCid: pgText(cid),
		MediaKind:    otherKind,
		ID:           self.ID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// The other half has not arrived: the wait is the answer.
		return
	}
	if err != nil {
		w.logger.Error("hunt the Live Photo's other half", "asset_id", assetID, "error", err)
		return
	}
	if match.IsComponent {
		// The counterpart already belongs to another pair (a shared
		// motion -- the same clip matched twice). Never steal it.
		return
	}

	// The still is the photo, whichever half arrived first; the motion
	// is the component.
	photoID, motionID := self.ID, match.ID
	photoLinked := self.LivePhotoVideoID.Valid
	if self.MediaKind == dbgen.MediaKindVideo {
		photoID, motionID = match.ID, self.ID
		photoLinked = match.LivePhotoVideoID.Valid
	}
	if photoLinked {
		// The still already carries a motion (the embedded extraction
		// won that race): the pair is formed, leave it alone.
		return
	}

	if err := queries.MarkAssetAsComponent(ctx, motionID); err != nil {
		w.logger.Error("hide the paired motion", "asset_id", assetID, "error", err)
		return
	}
	if err := queries.SetAssetLivePhotoVideo(ctx, dbgen.SetAssetLivePhotoVideoParams{
		ID:      photoID,
		VideoID: motionID,
	}); err != nil {
		w.logger.Error("link the paired halves", "asset_id", assetID, "error", err)
		return
	}

	// The wall's tile gains its type mark the moment this converges:
	// the still's row changed, and the detail refetch reads the link.
	w.bus.Publish("asset.updated", map[string]string{
		"assetId": uuid.UUID(photoID.Bytes).String(),
		"kind":    string(dbgen.MediaKindImage),
	})
}

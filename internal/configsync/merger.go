package configsync

import (
	"slices"
	"strings"

	"github.com/arturonavax/env/internal/domain"
)

type baseEditHunk struct {
	baseStart int // inclusive 0-based index in base
	baseEnd   int // exclusive 0-based index in base
	newLines  []string
}

// ThreeWayMerge performs a hunk-level diff3 3-way merge between base, local, and incoming text content.
func ThreeWayMerge(targetPath, base, local, incoming string) domain.ThreeWayDiff {
	diff := domain.ThreeWayDiff{
		TargetPath:      targetPath,
		BaseContent:     base,
		LocalContent:    local,
		IncomingContent: incoming,
	}

	// Fast paths when at most one side diverged or both converged identically.
	if local == incoming {
		diff.ConflictType = domain.ConflictNone
		diff.MergedContent = local
		diff.Resolution = domain.ResolutionAutoMerged
		return diff
	}
	if base == local {
		diff.ConflictType = domain.ConflictIncomingModified
		diff.MergedContent = incoming
		diff.Resolution = domain.ResolutionAutoMerged
		return diff
	}
	if base == incoming {
		diff.ConflictType = domain.ConflictLocalModified
		diff.MergedContent = local
		diff.Resolution = domain.ResolutionAutoMerged
		return diff
	}

	baseLines, baseNL := splitLogicalLines(base)
	localLines, localNL := splitLogicalLines(local)
	incLines, incNL := splitLogicalLines(incoming)

	if len(localLines) == 0 && len(baseLines) > 0 {
		diff.ConflictType = domain.ConflictDeletedLocal
		diff.HasConflict = true
		diff.Resolution = domain.ResolutionManual
		diff.MergedContent = formatConflict(local, incoming)
		return diff
	}
	if len(incLines) == 0 && len(baseLines) > 0 {
		diff.ConflictType = domain.ConflictDeletedIncoming
		diff.HasConflict = true
		diff.Resolution = domain.ResolutionManual
		diff.MergedContent = formatConflict(local, incoming)
		return diff
	}

	diff.ConflictType = domain.ConflictBothModified
	trailingNL := localNL || incNL || baseNL

	mergedLines, hasConflict := diff3MergeLines(baseLines, localLines, incLines)
	diff.HasConflict = hasConflict
	if hasConflict {
		diff.Resolution = domain.ResolutionManual
	} else {
		diff.Resolution = domain.ResolutionAutoMerged
	}

	if len(mergedLines) == 0 {
		diff.MergedContent = ""
		return diff
	}

	joined := strings.Join(mergedLines, "\n")
	if trailingNL {
		joined += "\n"
	}
	diff.MergedContent = joined
	return diff
}

func extractBaseHunks(base, side []string) []baseEditHunk {
	pairs := computeMatchedPairs(base, side)
	hunks := make([]baseEditHunk, 0, 8)

	prevB := -1
	prevS := -1

	for _, p := range pairs {
		if p.idxB > prevS+1 || p.idxA > prevB+1 {
			replacement := append([]string{}, side[prevS+1:p.idxB]...)
			hunks = append(hunks, baseEditHunk{
				baseStart: prevB + 1,
				baseEnd:   p.idxA,
				newLines:  replacement,
			})
		}
		prevB = p.idxA
		prevS = p.idxB
	}

	if len(side) > prevS+1 || len(base) > prevB+1 {
		replacement := append([]string{}, side[prevS+1:]...)
		hunks = append(hunks, baseEditHunk{
			baseStart: prevB + 1,
			baseEnd:   len(base),
			newLines:  replacement,
		})
	}

	return hunks
}

func hunksOverlap(a, b baseEditHunk) bool {
	aInsert := a.baseStart == a.baseEnd
	bInsert := b.baseStart == b.baseEnd

	switch {
	case aInsert && bInsert:
		return a.baseStart == b.baseStart
	case aInsert:
		return a.baseStart >= b.baseStart && a.baseStart <= b.baseEnd
	case bInsert:
		return b.baseStart >= a.baseStart && b.baseStart <= a.baseEnd
	default:
		return a.baseStart < b.baseEnd && b.baseStart < a.baseEnd
	}
}

func hunkTouchesSpan(h baseEditHunk, spanStart, spanEnd int) bool {
	if h.baseStart == h.baseEnd {
		return h.baseStart >= spanStart && h.baseStart <= spanEnd
	}
	if spanStart == spanEnd {
		return spanStart >= h.baseStart && spanStart <= h.baseEnd
	}
	return h.baseStart < spanEnd && spanStart < h.baseEnd
}

func applyHunksToSpan(base []string, spanStart, spanEnd int, hunks []baseEditHunk) []string {
	out := make([]string, 0, (spanEnd-spanStart)+8)
	cursor := spanStart
	for _, h := range hunks {
		if h.baseStart > cursor {
			out = append(out, base[cursor:h.baseStart]...)
		}
		out = append(out, h.newLines...)
		cursor = h.baseEnd
	}
	if spanEnd > cursor {
		out = append(out, base[cursor:spanEnd]...)
	}
	return out
}

func diff3MergeLines(base, local, incoming []string) ([]string, bool) {
	if slices.Equal(local, incoming) {
		return append([]string{}, local...), false
	}

	hunksL := extractBaseHunks(base, local)
	hunksR := extractBaseHunks(base, incoming)

	out := make([]string, 0, max(len(local), len(incoming))+8)
	hasConflict := false
	bPos := 0
	i := 0
	j := 0

	for i < len(hunksL) || j < len(hunksR) {
		if j >= len(hunksR) {
			h := hunksL[i]
			if h.baseStart > bPos {
				out = append(out, base[bPos:h.baseStart]...)
			}
			out = append(out, h.newLines...)
			bPos = h.baseEnd
			i++
			continue
		}

		if i >= len(hunksL) {
			h := hunksR[j]
			if h.baseStart > bPos {
				out = append(out, base[bPos:h.baseStart]...)
			}
			out = append(out, h.newLines...)
			bPos = h.baseEnd
			j++
			continue
		}

		hL := hunksL[i]
		hR := hunksR[j]

		if !hunksOverlap(hL, hR) {
			if hL.baseEnd < hR.baseStart || (hL.baseEnd == hR.baseStart && hL.baseStart < hR.baseStart) {
				if hL.baseStart > bPos {
					out = append(out, base[bPos:hL.baseStart]...)
				}
				out = append(out, hL.newLines...)
				bPos = hL.baseEnd
				i++
			} else {
				if hR.baseStart > bPos {
					out = append(out, base[bPos:hR.baseStart]...)
				}
				out = append(out, hR.newLines...)
				bPos = hR.baseEnd
				j++
			}
			continue
		}

		// Coalesce transitively overlapping hunks across both sides.
		spanStart := min(hL.baseStart, hR.baseStart)
		spanEnd := max(hL.baseEnd, hR.baseEnd)
		iStart := i
		jStart := j
		i++
		j++

		for {
			expanded := false
			for i < len(hunksL) && hunkTouchesSpan(hunksL[i], spanStart, spanEnd) {
				if hunksL[i].baseEnd > spanEnd {
					spanEnd = hunksL[i].baseEnd
				}
				i++
				expanded = true
			}
			for j < len(hunksR) && hunkTouchesSpan(hunksR[j], spanStart, spanEnd) {
				if hunksR[j].baseEnd > spanEnd {
					spanEnd = hunksR[j].baseEnd
				}
				j++
				expanded = true
			}
			if !expanded {
				break
			}
		}

		if spanStart > bPos {
			out = append(out, base[bPos:spanStart]...)
		}

		localSpan := applyHunksToSpan(base, spanStart, spanEnd, hunksL[iStart:i])
		incSpan := applyHunksToSpan(base, spanStart, spanEnd, hunksR[jStart:j])

		if slices.Equal(localSpan, incSpan) {
			out = append(out, localSpan...)
		} else {
			hasConflict = true
			out = append(out, "<<<<<<< LOCAL")
			out = append(out, localSpan...)
			out = append(out, "=======")
			out = append(out, incSpan...)
			out = append(out, ">>>>>>> INCOMING")
		}

		bPos = spanEnd
	}

	if bPos < len(base) {
		out = append(out, base[bPos:]...)
	}

	return out, hasConflict
}

func formatConflict(local, incoming string) string {
	var sb strings.Builder
	sb.WriteString("<<<<<<< LOCAL\n")
	sb.WriteString(local)
	if local != "" && !strings.HasSuffix(local, "\n") {
		sb.WriteString("\n")
	}
	sb.WriteString("=======\n")
	sb.WriteString(incoming)
	if incoming != "" && !strings.HasSuffix(incoming, "\n") {
		sb.WriteString("\n")
	}
	sb.WriteString(">>>>>>> INCOMING\n")
	return sb.String()
}

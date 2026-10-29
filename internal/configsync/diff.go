package configsync

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/arturonavax/env/internal/security"
)

// DiffStatus categorizes how a file differs between two trees.
type DiffStatus string

const (
	DiffModified DiffStatus = "modified"
	DiffAdded    DiffStatus = "added"
	DiffRemoved  DiffStatus = "removed"
)

// FileDiff represents the unified diff for one tracked file between two paths.
type FileDiff struct {
	RelPath string     `json:"rel_path"`
	PathA   string     `json:"path_a"`
	PathB   string     `json:"path_b"`
	Status  DiffStatus `json:"status"`
	Patch   string     `json:"patch"`
}

type changeGroup struct {
	startOp int
	endOp   int // exclusive index into ops
}

// UnifiedDiff generates a standard unified diff between textA and textB with contextLines surrounding lines,
// using Myers O(ND) line diffing in pure Go without external processes.
func UnifiedDiff(labelA, labelB, textA, textB string, contextLines int) string {
	if textA == textB {
		return ""
	}
	if contextLines < 0 {
		contextLines = 0
	}

	linesA, _ := splitLogicalLines(textA)
	linesB, _ := splitLogicalLines(textB)
	ops := computeEditScript(linesA, linesB)

	// Identify non-equal op spans.
	rawChanges := make([]changeGroup, 0, 8)
	idx := 0
	for idx < len(ops) {
		if ops[idx].kind == opEqual {
			idx++
			continue
		}
		start := idx
		for idx < len(ops) && ops[idx].kind != opEqual {
			idx++
		}
		rawChanges = append(rawChanges, changeGroup{startOp: start, endOp: idx})
	}
	if len(rawChanges) == 0 {
		return ""
	}

	// Expand by contextLines and merge adjacent/overlapping hunks.
	hunks := make([]changeGroup, 0, len(rawChanges))
	for _, cg := range rawChanges {
		hStart := max(0, cg.startOp-contextLines)
		hEnd := min(len(ops), cg.endOp+contextLines)
		if len(hunks) > 0 && hStart <= hunks[len(hunks)-1].endOp {
			hunks[len(hunks)-1].endOp = max(hunks[len(hunks)-1].endOp, hEnd)
			continue
		}
		hunks = append(hunks, changeGroup{startOp: hStart, endOp: hEnd})
	}

	var sb strings.Builder
	sb.WriteString("--- ")
	sb.WriteString(labelA)
	sb.WriteByte('\n')
	sb.WriteString("+++ ")
	sb.WriteString(labelB)
	sb.WriteByte('\n')

	for _, h := range hunks {
		sliceOps := ops[h.startOp:h.endOp]
		startA, countA, startB, countB := hunkLineCoordinates(sliceOps)

		fmt.Fprintf(&sb, "@@ -%s +%s @@\n", formatRange(startA, countA), formatRange(startB, countB))
		for _, op := range sliceOps {
			switch op.kind {
			case opEqual:
				sb.WriteByte(' ')
				sb.WriteString(op.text)
				sb.WriteByte('\n')
			case opDelete:
				sb.WriteByte('-')
				sb.WriteString(op.text)
				sb.WriteByte('\n')
			case opInsert:
				sb.WriteByte('+')
				sb.WriteString(op.text)
				sb.WriteByte('\n')
			}
		}
	}

	return sb.String()
}

func hunkLineCoordinates(sliceOps []diffOp) (startA, countA, startB, countB int) {
	firstA := -1
	firstB := -1

	for _, op := range sliceOps {
		switch op.kind {
		case opEqual:
			if firstA == -1 {
				firstA = op.idxA + 1
			}
			if firstB == -1 {
				firstB = op.idxB + 1
			}
			countA++
			countB++
		case opDelete:
			if firstA == -1 {
				firstA = op.idxA + 1
			}
			if firstB == -1 {
				firstB = op.idxB + 1
			}
			countA++
		case opInsert:
			if firstA == -1 {
				firstA = op.idxA + 1
			}
			if firstB == -1 {
				firstB = op.idxB + 1
			}
			countB++
		}
	}

	if countA == 0 {
		startA = 0
	} else {
		startA = max(1, firstA)
	}
	if countB == 0 {
		startB = 0
	} else {
		startB = max(1, firstB)
	}
	return startA, countA, startB, countB
}

func formatRange(start, count int) string {
	if count == 1 {
		return fmt.Sprintf("%d", start)
	}
	return fmt.Sprintf("%d,%d", start, count)
}

// DiffTree recursively compares two directories (or two files) while honoring glob exclusions,
// absorbing utils/diff_snapshots.sh in pure Go.
func DiffTree(pathA, pathB string, excludes []string) ([]FileDiff, error) {
	if strings.TrimSpace(pathA) == "" || strings.TrimSpace(pathB) == "" {
		return nil, security.ErrInvalidPath
	}

	infoA, errA := os.Stat(pathA)
	if errA != nil && !os.IsNotExist(errA) {
		return nil, fmt.Errorf("stat %q: %w", pathA, errA)
	}
	infoB, errB := os.Stat(pathB)
	if errB != nil && !os.IsNotExist(errB) {
		return nil, fmt.Errorf("stat %q: %w", pathB, errB)
	}

	existsA := errA == nil
	existsB := errB == nil
	if !existsA && !existsB {
		return []FileDiff{}, nil
	}

	// Single-file comparison when neither existing path is a directory.
	isDirA := existsA && infoA.IsDir()
	isDirB := existsB && infoB.IsDir()
	if !isDirA && !isDirB {
		diff, ok, err := diffSingleFilePair(filepath.Base(pathA), pathA, pathB, existsA, existsB)
		if err != nil {
			return nil, err
		}
		if !ok {
			return []FileDiff{}, nil
		}
		return []FileDiff{diff}, nil
	}

	filesA := map[string]string{}
	if isDirA {
		var err error
		filesA, err = collectTreeFiles(pathA, excludes)
		if err != nil {
			return nil, err
		}
	}

	filesB := map[string]string{}
	if isDirB {
		var err error
		filesB, err = collectTreeFiles(pathB, excludes)
		if err != nil {
			return nil, err
		}
	}

	relSet := make(map[string]bool, len(filesA)+len(filesB))
	for rel := range filesA {
		relSet[rel] = true
	}
	for rel := range filesB {
		relSet[rel] = true
	}

	relPaths := make([]string, 0, len(relSet))
	for rel := range relSet {
		relPaths = append(relPaths, rel)
	}
	slices.Sort(relPaths)

	results := []FileDiff{}
	for _, rel := range relPaths {
		fileA, inA := filesA[rel]
		fileB, inB := filesB[rel]
		if !inA {
			fileA = filepath.Join(pathA, filepath.FromSlash(rel))
		}
		if !inB {
			fileB = filepath.Join(pathB, filepath.FromSlash(rel))
		}

		fd, changed, err := diffSingleFilePair(rel, fileA, fileB, inA, inB)
		if err != nil {
			return nil, err
		}
		if changed {
			results = append(results, fd)
		}
	}

	return results, nil
}

func collectTreeFiles(root string, excludes []string) (map[string]string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	out := map[string]string{}
	err = filepath.WalkDir(absRoot, func(curr string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(absRoot, curr)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if matchesAnyExclude(rel, d.Name(), excludes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		out[filepath.ToSlash(rel)] = curr
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func diffSingleFilePair(rel, fileA, fileB string, inA, inB bool) (FileDiff, bool, error) {
	var textA string
	if inA {
		raw, err := os.ReadFile(fileA)
		if err != nil {
			return FileDiff{}, false, fmt.Errorf("reading %q: %w", fileA, err)
		}
		textA = string(raw)
	}

	var textB string
	if inB {
		raw, err := os.ReadFile(fileB)
		if err != nil {
			return FileDiff{}, false, fmt.Errorf("reading %q: %w", fileB, err)
		}
		textB = string(raw)
	}

	if inA && inB && textA == textB {
		return FileDiff{}, false, nil
	}

	status := DiffModified
	switch {
	case !inA && inB:
		status = DiffAdded
	case inA && !inB:
		status = DiffRemoved
	}

	patch := UnifiedDiff(fileA, fileB, textA, textB, 3)
	if patch == "" && status == DiffModified {
		return FileDiff{}, false, nil
	}

	return FileDiff{
		RelPath: rel,
		PathA:   fileA,
		PathB:   fileB,
		Status:  status,
		Patch:   patch,
	}, true, nil
}

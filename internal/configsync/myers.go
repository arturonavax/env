package configsync

import "strings"

const maxMyersEditDistance = 4096

type editKind uint8

const (
	opEqual editKind = iota + 1
	opDelete
	opInsert
)

type diffOp struct {
	kind editKind
	text string
	idxA int // 0-based index in A (valid for opEqual and opDelete)
	idxB int // 0-based index in B (valid for opEqual and opInsert)
}

type matchedPair struct {
	idxA int
	idxB int
}

type vSnapshot struct {
	offset int
	data   []int
}

func (vs vSnapshot) get(k int) int {
	return vs.data[vs.offset+k]
}

// splitLogicalLines normalizes CRLF to LF and splits text into lines without trailing newline delimiters,
// returning whether the normalized input ended with a trailing newline.
func splitLogicalLines(s string) ([]string, bool) {
	if s == "" {
		return []string{}, false
	}
	norm := strings.ReplaceAll(s, "\r\n", "\n")
	trailingNL := strings.HasSuffix(norm, "\n")
	if trailingNL {
		norm = norm[:len(norm)-1]
	}
	return strings.Split(norm, "\n"), trailingNL
}

// computeMatchedPairs extracts strictly increasing common line index pairs from the Myers shortest edit script.
func computeMatchedPairs(a, b []string) []matchedPair {
	ops := computeEditScript(a, b)
	pairs := make([]matchedPair, 0, min(len(a), len(b)))
	for _, op := range ops {
		if op.kind == opEqual {
			pairs = append(pairs, matchedPair{idxA: op.idxA, idxB: op.idxB})
		}
	}
	return pairs
}

// computeEditScript computes a shortest edit script transforming slice a into slice b using Eugene Myers' O(ND) algorithm.
func computeEditScript(a, b []string) []diffOp {
	lenA := len(a)
	lenB := len(b)
	if lenA == 0 && lenB == 0 {
		return []diffOp{}
	}

	prefix := 0
	for prefix < lenA && prefix < lenB && a[prefix] == b[prefix] {
		prefix++
	}

	suffix := 0
	for suffix < (lenA-prefix) && suffix < (lenB-prefix) && a[lenA-1-suffix] == b[lenB-1-suffix] {
		suffix++
	}

	aMid := a[prefix : lenA-suffix]
	bMid := b[prefix : lenB-suffix]
	midOps := myersMiddle(aMid, bMid, prefix)

	totalCap := prefix + len(midOps) + suffix
	ops := make([]diffOp, 0, totalCap)

	for i := range prefix {
		ops = append(ops, diffOp{
			kind: opEqual,
			text: a[i],
			idxA: i,
			idxB: i,
		})
	}

	ops = append(ops, midOps...)

	startSuffixA := lenA - suffix
	startSuffixB := lenB - suffix
	for i := range suffix {
		ops = append(ops, diffOp{
			kind: opEqual,
			text: a[startSuffixA+i],
			idxA: startSuffixA + i,
			idxB: startSuffixB + i,
		})
	}

	return ops
}

func myersMiddle(a, b []string, baseOffset int) []diffOp {
	n := len(a)
	m := len(b)
	if n == 0 && m == 0 {
		return []diffOp{}
	}
	if n == 0 {
		ops := make([]diffOp, 0, m)
		for j, line := range b {
			ops = append(ops, diffOp{
				kind: opInsert,
				text: line,
				idxA: baseOffset,
				idxB: baseOffset + j,
			})
		}
		return ops
	}
	if m == 0 {
		ops := make([]diffOp, 0, n)
		for i, line := range a {
			ops = append(ops, diffOp{
				kind: opDelete,
				text: line,
				idxA: baseOffset + i,
				idxB: baseOffset,
			})
		}
		return ops
	}

	maxD := n + m
	vOffset := maxD
	v := make([]int, 2*maxD+1)

	// trace[step] holds v values at the end of (step-1) over indices [-(step-1)..(step-1)].
	trace := make([]vSnapshot, 1, min(maxD+1, 64))
	finalD := -1

	limitD := min(maxD, maxMyersEditDistance)
	for d := 0; d <= limitD; d++ {
		if d > 0 {
			span := d - 1
			snap := make([]int, 2*span+1)
			copy(snap, v[vOffset-span:vOffset+span+1])
			trace = append(trace, vSnapshot{
				offset: span,
				data:   snap,
			})
		}

		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[vOffset+k-1] < v[vOffset+k+1]) {
				x = v[vOffset+k+1]
			} else {
				x = v[vOffset+k-1] + 1
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[vOffset+k] = x
			if x >= n && y >= m {
				finalD = d
				break
			}
		}
		if finalD != -1 {
			break
		}
	}

	// Fallback if edit distance exceeded cap on huge divergent inputs.
	if finalD == -1 {
		ops := make([]diffOp, 0, n+m)
		for i, line := range a {
			ops = append(ops, diffOp{kind: opDelete, text: line, idxA: baseOffset + i, idxB: baseOffset})
		}
		for j, line := range b {
			ops = append(ops, diffOp{kind: opInsert, text: line, idxA: baseOffset + n, idxB: baseOffset + j})
		}
		return ops
	}

	revOps := make([]diffOp, 0, n+m)
	x := n
	y := m

	for step := finalD; step > 0; step-- {
		vPrev := trace[step]
		k := x - y

		var prevK int
		if k == -step || (k != step && vPrev.get(k-1) < vPrev.get(k+1)) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}

		prevX := vPrev.get(prevK)
		prevY := prevX - prevK

		for x > prevX && y > prevY {
			x--
			y--
			revOps = append(revOps, diffOp{
				kind: opEqual,
				text: a[x],
				idxA: baseOffset + x,
				idxB: baseOffset + y,
			})
		}

		if prevK == k+1 {
			y--
			revOps = append(revOps, diffOp{
				kind: opInsert,
				text: b[y],
				idxA: baseOffset + x,
				idxB: baseOffset + y,
			})
		} else {
			x--
			revOps = append(revOps, diffOp{
				kind: opDelete,
				text: a[x],
				idxA: baseOffset + x,
				idxB: baseOffset + y,
			})
		}
	}

	for x > 0 && y > 0 {
		x--
		y--
		revOps = append(revOps, diffOp{
			kind: opEqual,
			text: a[x],
			idxA: baseOffset + x,
			idxB: baseOffset + y,
		})
	}

	// Reverse revOps in place.
	for left, right := 0, len(revOps)-1; left < right; left, right = left+1, right-1 {
		revOps[left], revOps[right] = revOps[right], revOps[left]
	}

	return revOps
}

// Portions ported from Grammalecte's graphspell/str_transform.py by Olivier R.
// Distributed under GNU GPL version 3; see the repository LICENSE.
package strtransform

import "github.com/benoit-pereira-da-silva/gogrammalecte/graphspell/charplayer"

// DistanceDamerauLevenshtein computes the upstream adjacent-transposition edit
// distance over Unicode code points.
func DistanceDamerauLevenshtein(first, second string) int {
	return int(editDistance([]rune(first), []rune(second), false))
}

// DistanceDamerauLevenshteinX uses the upstream character substitution weights
// for words of at most forty code points; longer words use unit weights.
// Scratch state is local to a call, allowing concurrent use.
func DistanceDamerauLevenshteinX(first, second string) float64 {
	a, b := []rune(first), []rune(second)
	return editDistance(a, b, len(a) <= 40 && len(b) <= 40)
}
func editDistance(a, b []rune, weighted bool) float64 {
	rows := make([][]float64, len(a)+1)
	for i := range rows {
		rows[i] = make([]float64, len(b)+1)
		rows[i][0] = float64(i)
	}
	for j := range rows[0] {
		rows[0][j] = float64(j)
	}
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1.0
			if weighted {
				cost = charplayer.DistanceBetweenChars(a[i-1], b[j-1])
			} else if a[i-1] == b[j-1] {
				cost = 0
			}
			rows[i][j] = min(rows[i-1][j]+1, rows[i][j-1]+1, rows[i-1][j-1]+cost)
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				rows[i][j] = min(rows[i][j], rows[i-2][j-2]+cost)
			}
		}
	}
	return rows[len(a)][len(b)]
}

// DistanceJaroWinkler reproduces the reference's similarity score, including
// its unusual index order and character-table adjustment. The optional boost
// threshold defaults to 0.666. This is not a generic Jaro-Winkler implementation.
func DistanceJaroWinkler(first, second string, boost ...float64) float64 {
	if first == second {
		return 1
	}
	threshold := 0.666
	if len(boost) > 0 {
		threshold = boost[0]
	}
	a, b := []rune(first), []rune(second)
	n1, n2 := len(a), len(b)
	nmax := max(n1, n2)
	flags1, flags2 := make([]int, nmax), make([]int, nmax)
	search := nmax/2 - 1
	nmin := min(n1, n2)
	common := 0
	jLast := 0
	for i := 0; i < n1; i++ {
		low := 0
		if i >= search {
			low = i - search
		}
		high := min(i+search, n2-1)
		for j := low; j <= high; j++ {
			jLast = j
			if flags2[j] != 1 && runeSlice(a, j) == runeSlice(b, i) {
				flags1[j] = 1
				flags2[i] = 1
				common++
				break
			}
		}
	}
	if common == 0 {
		return 0
	}
	k, trans := 0, 0
	for i := 0; i < n1; i++ {
		if flags1[i] == 1 {
			for j := k; j < n2; j++ {
				jLast = j
				if flags2[j] == 1 {
					k = j + 1
					break
				}
			}
			if a[i] != b[jLast] {
				trans++
			}
		}
	}
	trans /= 2
	similarity := 0.0
	if nmin > common {
		for i := 0; i < n1; i++ {
			if flags1[i] == 0 {
				for j := 0; j < n2; j++ {
					if flags2[j] == 0 {
						if v, ok := charplayer.KnownDistance(a[i], b[j]); ok {
							similarity += v
							flags2[j] = 2
							break
						}
					}
				}
			}
		}
	}
	sim := similarity/10 + float64(common)
	weight := (sim/float64(n1) + sim/float64(n2) + float64(common-trans)/float64(common)) / 3
	if weight > threshold {
		i := 0
		for i < min(4, nmin) && a[i] == b[i] {
			i++
		}
		if i > 0 {
			weight += float64(i) * 0.1 * (1 - weight)
		}
		if nmin > 4 && common > i+1 && 2*common >= nmin+i {
			weight += (1 - weight) * (float64(common-i-1) / float64(n1*n2-i*2+2))
		}
	}
	return weight
}
func runeSlice(s []rune, i int) string {
	if i < 0 || i >= len(s) {
		return ""
	}
	return string(s[i])
}

// DistanceSift4 computes the reference Sift4 score. Its optional search offset
// defaults to five, preserving the original algorithm's traversal order.
func DistanceSift4(first, second string, maxOffsets ...int) int {
	a, b := []rune(first), []rune(second)
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	maxOffset := 5
	if len(maxOffsets) > 0 {
		maxOffset = maxOffsets[0]
	}
	type offset struct {
		a, b       int
		transposed bool
	}
	offsets := []offset{}
	i1, i2, largest, local, trans := 0, 0, 0, 0, 0
	for i1 < len(a) && i2 < len(b) {
		if a[i1] == b[i2] {
			local++
			isTrans := false
			i := 0
			for i < len(offsets) {
				item := &offsets[i]
				if i1 <= item.a || i2 <= item.b {
					isTrans = abs(i2-i1) >= abs(item.b-item.a)
					if isTrans {
						trans++
					} else if !item.transposed {
						item.transposed = true
						trans++
					}
					break
				}
				if i1 > item.b && i2 > item.a {
					offsets = append(offsets[:i], offsets[i+1:]...)
				} else {
					i++
				}
			}
			offsets = append(offsets, offset{i1, i2, isTrans})
		} else {
			largest += local
			local = 0
			if i1 != i2 {
				i1 = min(i1, i2)
				i2 = i1
			}
			for i := 0; i < maxOffset; i++ {
				if i1+i >= len(a) && i2+i >= len(b) {
					break
				}
				if i1+i < len(a) && a[i1+i] == b[i2] {
					i1 += i - 1
					i2--
					break
				}
				if i2+i < len(b) && a[i1] == b[i2+i] {
					i1--
					i2 += i - 1
					break
				}
			}
		}
		i1++
		i2++
		if i1 >= len(a) || i2 >= len(b) {
			largest += local
			local = 0
			i1 = min(i1, i2)
			i2 = i1
		}
	}
	return max(len(a), len(b)) - largest - local + trans
}
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

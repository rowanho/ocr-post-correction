package alignment

import (
	"math"
)

func reverseRune(r []rune) []rune {
	f := make([]rune, len(r))
	copy(f, r)
	for i, j := 0, len(f)-1; i < j; i, j = i+1, j-1 {
		f[i], f[j] = f[j], f[i]
	}
	return f
}

func reverseInt(r []int) []int {
	f := append([]int{}, r...)
	for i, j := 0, len(f)-1; i < j; i, j = i+1, j-1 {
		f[i], f[j] = f[j], f[i]
	}
	return f
}

func nwScore(matchReward int, gapCost int, a []rune, b []rune) []int {

	score := make([][]int, 2)

	for i := 0; i < 2; i++ {
		score[i] = make([]int, len(b)+1)
	}

	for j := 1; j < len(b)+1; j++ {
		score[0][j] = score[0][j-1] - gapCost
	}
	for i := 1; i < len(a)+1; i++ {
		score[1][0] = score[0][0] - gapCost
		for j := 1; j < len(b)+1; j++ {
			var match int
			if a[i-1] == b[j-1] {
				match = score[0][j-1] + matchReward
			} else {
				match = score[0][j-1] - matchReward
			}
			delete := score[0][j] - gapCost
			insert := score[1][j-1] - gapCost
			score[1][j] = Max(match, Max(delete, insert))
		}
		for j := 1; j < len(b); j++ {
			score[0][j] = score[1][j]
		}
	}

	return score[1]
}

func swScore(matchReward int, gapCost int, a []rune, b []rune) (int, int) {

	max_i := 0
	max_j := 0
	max_val := 0
	score := make([][]int, 2)

	lenA := len(a)
	lenB := len(b)
	for i := 0; i < 2; i++ {
		score[i] = make([]int, len(b)+1)
	}

	for i := 1; i < lenA+1; i++ {
		for j := 1; j < lenB+1; j++ {
			var match int
			if a[i-1] == b[j-1] {
				match = score[0][j-1] + matchReward
			} else {
				match = score[0][j-1] - matchReward
			}
			delete := score[0][j] - gapCost
			insert := score[1][j-1] - gapCost
			score[1][j] = Max(match, Max(delete, Max(0, insert)))
			if score[1][j] > max_val {
				max_val = score[1][j]
				max_i = i
				max_j = j
			}
		}

		for j := 1; j < lenB; j++ {
			score[0][j] = score[1][j]
		}

	}
	return max_i - 1, max_j - 1
}

func hirschberg(matchReward int, gapCost int, a []rune, b []rune, offsetA int, offsetB int) (int, []int, []int) {
	lenA := len(a)
	lenB := len(b)

	if lenA == 0 {
		score := 0
		for i := 0; i < lenB; i++ {
			score -= gapCost
		}
		return score, []int{}, []int{}
	} else if lenB == 0 {
		score := 0
		for i := 0; i < lenA; i++ {
			score -= gapCost
		}
		return score, []int{}, []int{}
	} else if lenA == 1 || lenB == 1 {
		score, nwResA, nwResB := NeedlemanWunsch(matchReward, gapCost, a, b)
		listA := make([]int, len(nwResA))
		listB := make([]int, len(nwResB))
		for i := 0; i < len(nwResA); i++ {
			listA[i] = nwResA[i] + offsetA
		}
		for i := 0; i < len(nwResB); i++ {
			listB[i] = nwResB[i] + offsetB
		}
		return score, listA, listB
	}

	midA := lenA / 2

	lastlineLeft := nwScore(matchReward, gapCost, a[0:midA], b)
	revASlice := reverseRune(a[midA:])
	revB := reverseRune(b)
	lastlineRight := nwScore(matchReward, gapCost, revASlice, revB)
	lastlineRight = reverseInt(lastlineRight)

	max := math.MinInt32
	maxIndice := 0
	for i := 0; i < len(lastlineLeft); i++ {
		if max <= lastlineLeft[i]+lastlineRight[i] {
			max = lastlineLeft[i] + lastlineRight[i]
			maxIndice = i
		}
	}
	midB := maxIndice
	firstScore, firstRes1, firstRes2 := hirschberg(matchReward, gapCost, a[0:midA], b[0:midB], offsetA, offsetB)
	secondScore, secondRes1, secondRes2 := hirschberg(matchReward, gapCost, a[midA:], b[midB:], midA+offsetA, midB+offsetB)

	score := firstScore + secondScore
	aIndices := append(firstRes1, secondRes1...)
	bIndices := append(firstRes2, secondRes2...)

	return score, aIndices, bIndices

}


// A banded implementation of the Smith Waterman algorithm 
// Uses O(n) space, and O(mn) time
func SmithWaterman(matchReward int, gapCost int, a []rune, b []rune) (int, []int, []int) {
	if len(a) == 0 || len(b) == 0 {
		return 0, []int{}, []int{}
	}
	endA, endB := swScore(matchReward, gapCost, a, b)
	revA := reverseRune(a)
	revB := reverseRune(b)
	revStartA, revStartB := swScore(matchReward, gapCost, revA, revB)

	if revStartA == 0 || revStartB == 0 {
		return 0, []int{}, []int{}
	}
	startA := len(a) - 1 - revStartA
	startB := len(b) - 1 - revStartB
	if startA > endA || startB > endB {
		return 0, []int{}, []int{}
	}

	return hirschberg(matchReward, gapCost, a[startA:endA+1], b[startB:endB+1], startA, startB)
}

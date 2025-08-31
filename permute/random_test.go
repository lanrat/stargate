package permute

import (
	"fmt"
	"math/big"
	"sort"
	"testing"
)

func TestRandomParallelIterator_NonDeterministic(t *testing.T) {
	t.Parallel()

	low := big.NewInt(0)
	high := big.NewInt(100)

	// Create two iterators with identical parameters
	iter1, err := NewRandomParallelIterator(low, high)
	if err != nil {
		t.Fatalf("failed to create first iterator: %v", err)
	}

	iter2, err := NewRandomParallelIterator(low, high)
	if err != nil {
		t.Fatalf("failed to create second iterator: %v", err)
	}

	// Collect sequences from both iterators
	var seq1, seq2 []int64

	for {
		num1, ok1 := iter1.Next()
		if !ok1 {
			break
		}
		seq1 = append(seq1, num1.Int64())
	}

	for {
		num2, ok2 := iter2.Next()
		if !ok2 {
			break
		}
		seq2 = append(seq2, num2.Int64())
	}

	// Both sequences should have the same length
	if len(seq1) != len(seq2) {
		t.Errorf("sequences have different lengths: %d vs %d", len(seq1), len(seq2))
	}

	// Both sequences should contain the same elements (just verify a few)
	expectedLen := 100 // [0, 100) exclusive
	if len(seq1) != expectedLen {
		t.Errorf("expected %d elements, got %d", expectedLen, len(seq1))
	}

	// Sequences should be different (very high probability)
	identical := true
	for i := 0; i < len(seq1) && i < len(seq2); i++ {
		if seq1[i] != seq2[i] {
			identical = false
			break
		}
	}

	if identical {
		t.Error("sequences are identical - randomization may not be working")
	}

	// Both sequences should contain all numbers in range when sorted
	sort.Slice(seq1, func(i, j int) bool { return seq1[i] < seq1[j] })
	sort.Slice(seq2, func(i, j int) bool { return seq2[i] < seq2[j] })

	for i := 0; i < 100; i++ {
		if seq1[i] != int64(i) {
			t.Errorf("seq1 missing or has wrong number at position %d: got %d", i, seq1[i])
		}
		if seq2[i] != int64(i) {
			t.Errorf("seq2 missing or has wrong number at position %d: got %d", i, seq2[i])
		}
	}
}

func TestRandomParallelIterator_BoundsChecking(t *testing.T) {
	t.Parallel()

	low := big.NewInt(50)
	high := big.NewInt(150)

	iter, err := NewRandomParallelIterator(low, high)
	if err != nil {
		t.Fatalf("failed to create iterator: %v", err)
	}

	// Check that all generated numbers are within bounds
	count := 0
	for {
		num, ok := iter.Next()
		if !ok {
			break
		}

		if num.Cmp(low) < 0 || num.Cmp(high) >= 0 {
			t.Errorf("number %s out of range [%s, %s)", num, low, high)
		}
		count++
	}

	expectedCount := 100 // [50, 150) exclusive
	if count != expectedCount {
		t.Errorf("expected %d numbers, got %d", expectedCount, count)
	}
}

func TestRandomParallelIterator_InvalidRange(t *testing.T) {
	t.Parallel()

	// Test invalid range where low > high
	low := big.NewInt(100)
	high := big.NewInt(50)

	_, err := NewRandomParallelIterator(low, high)
	if err == nil {
		t.Error("expected error for invalid range, got nil")
	}
}

func TestRandomParallelIterator_SingleElement(t *testing.T) {
	t.Parallel()

	// Test range with single element
	low := big.NewInt(42)
	high := big.NewInt(43)

	iter, err := NewRandomParallelIterator(low, high)
	if err != nil {
		t.Fatalf("failed to create iterator: %v", err)
	}

	// Should return exactly one number
	num, ok := iter.Next()
	if !ok {
		t.Error("expected one number, got none")
	}

	if num.Cmp(big.NewInt(42)) != 0 {
		t.Errorf("expected 42, got %s", num)
	}

	// Second call should return false
	_, ok = iter.Next()
	if ok {
		t.Error("expected no more numbers after exhaustion")
	}
}

func TestRandomUniqueRand_NonDeterministic(t *testing.T) {
	t.Parallel()

	low := big.NewInt(0)
	high := big.NewInt(100)

	// Create two iterators with identical parameters
	iter1, err := NewRandomUniqueRand(low, high)
	if err != nil {
		t.Fatalf("failed to create first iterator: %v", err)
	}

	iter2, err := NewRandomUniqueRand(low, high)
	if err != nil {
		t.Fatalf("failed to create second iterator: %v", err)
	}

	// Compare first 20 values
	differences := 0
	for i := 0; i < 20; i++ {
		idx := big.NewInt(int64(i))

		num1 := iter1.NextAt(idx)
		num2 := iter2.NextAt(idx)

		// Both should be in range
		if num1.Cmp(low) < 0 || num1.Cmp(high) > 0 {
			t.Errorf("iter1 number %s out of range at index %d", num1, i)
		}
		if num2.Cmp(low) < 0 || num2.Cmp(high) > 0 {
			t.Errorf("iter2 number %s out of range at index %d", num2, i)
		}

		// Count differences
		if num1.Cmp(num2) != 0 {
			differences++
		}
	}

	// Should have some differences (high probability with random offsets)
	if differences == 0 {
		t.Error("all values identical - randomization may not be working")
	}
}

func TestRandomUniqueRand_BoundsChecking(t *testing.T) {
	t.Parallel()

	low := big.NewInt(1000)
	high := big.NewInt(2000)

	iter, err := NewRandomUniqueRand(low, high)
	if err != nil {
		t.Fatalf("failed to create iterator: %v", err)
	}

	// Test various indices
	for i := 0; i < 100; i++ {
		idx := big.NewInt(int64(i))
		num := iter.NextAt(idx)

		if num.Cmp(low) < 0 || num.Cmp(high) >= 0 {
			t.Errorf("number %s out of range [%s, %s] at index %d", num, low, high, i)
		}
	}
}

func TestRandomIterators_HelperMethods(t *testing.T) {
	t.Parallel()

	low := big.NewInt(10)
	high := big.NewInt(20)

	// Test RandomParallelIterator methods
	rpi, err := NewRandomParallelIterator(low, high)
	if err != nil {
		t.Fatalf("failed to create RandomParallelIterator: %v", err)
	}

	if rpi.Size().Cmp(big.NewInt(10)) != 0 {
		t.Errorf("expected size 10, got %s", rpi.Size())
	}

	if rpi.Low().Cmp(low) != 0 {
		t.Errorf("expected low %s, got %s", low, rpi.Low())
	}

	if rpi.High().Cmp(high) != 0 {
		t.Errorf("expected high %s, got %s", high, rpi.High())
	}

	// Test RandomUniqueRand methods
	ru, err := NewRandomUniqueRand(low, high)
	if err != nil {
		t.Fatalf("failed to create RandomUniqueRand: %v", err)
	}

	if ru.Size().Cmp(big.NewInt(10)) != 0 {
		t.Errorf("expected size 10, got %s", ru.Size())
	}

	if ru.Low().Cmp(low) != 0 {
		t.Errorf("expected low %s, got %s", low, ru.Low())
	}

	if ru.High().Cmp(high) != 0 {
		t.Errorf("expected high %s, got %s", high, ru.High())
	}
}

func BenchmarkRandomParallelIterator(b *testing.B) {
	iter, _ := NewRandomParallelIterator(big.NewInt(0), big.NewInt(1<<20+1))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			iter.Next()
		}
	})
}

func BenchmarkRandomUniqueRand(b *testing.B) {
	iter, _ := NewRandomUniqueRand(big.NewInt(0), big.NewInt(1<<20+1))
	index := big.NewInt(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iter.NextAt(index)
		index.Add(index, big.NewInt(1))
		if index.Cmp(iter.Size()) >= 0 {
			index.SetInt64(0)
		}
	}
}

func ExampleRandomParallelIterator() {
	// Create two random iterators with the same range
	iter1, _ := NewRandomParallelIterator(big.NewInt(0), big.NewInt(5))
	iter2, _ := NewRandomParallelIterator(big.NewInt(0), big.NewInt(5))

	// They will visit the same numbers but in different orders
	fmt.Println("Iterator 1:")
	for {
		num, ok := iter1.Next()
		if !ok {
			break
		}
		fmt.Printf("%s ", num)
	}

	fmt.Println("\nIterator 2:")
	for {
		num, ok := iter2.Next()
		if !ok {
			break
		}
		fmt.Printf("%s ", num)
	}

	// Output will vary between runs, but each iterator will output 0,1,2,3,4 in some order
	// Example output:
	// Iterator 1: 2 0 4 1 3
	// Iterator 2: 1 3 0 4 2
}

func TestRandomParallelIterator_NoSequentialRuns(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		size *big.Int
	}{
		{"size_255", big.NewInt(255)},
		{"size_256", big.NewInt(256)},
		{"size_1000", big.NewInt(1000)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			iter, err := NewRandomParallelIterator(big.NewInt(0), tc.size)
			if err != nil {
				t.Fatal(err)
			}

			prev, _ := iter.Next()
			sequentialCount := 0
			maxSequential := 0

			// Check first 100 values or size, whichever is smaller
			checkCount := 100
			if tc.size.Cmp(big.NewInt(100)) < 0 {
				checkCount = int(tc.size.Int64())
			}

			for i := 1; i < checkCount; i++ {
				curr, ok := iter.Next()
				if !ok {
					break
				}

				// Check if current is sequential to previous
				diff := new(big.Int).Sub(curr, prev)
				if diff.Cmp(big.NewInt(1)) == 0 {
					sequentialCount++
					if sequentialCount > maxSequential {
						maxSequential = sequentialCount
					}
				} else {
					sequentialCount = 0
				}

				prev = curr
			}

			// Allow at most 2 sequential numbers in a row by chance
			if maxSequential > 2 {
				t.Errorf("Found %d sequential numbers in a row for size %s, indicating non-random behavior",
					maxSequential+1, tc.size.String())
			}
		})
	}
}

func ExampleRandomUniqueRand() {
	// Create a random iterator
	iter, _ := NewRandomUniqueRand(big.NewInt(10), big.NewInt(15))

	// Access values at specific indices - will be randomized per instance
	for i := 0; i < 5; i++ {
		num := iter.NextAt(big.NewInt(int64(i)))
		fmt.Printf("Index %d: %s\n", i, num)
	}

	// Output will vary between runs
	// Example output:
	// Index 0: 13
	// Index 1: 11
	// Index 2: 14
	// Index 3: 10
	// Index 4: 12
}

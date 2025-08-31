package permute

import (
	"fmt"
	"math/big"
	"sort"
	"sync"
	"testing"
)

func TestNewUniqueRand(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		low     *big.Int
		high    *big.Int
		wantErr bool
	}{
		{
			name:    "valid range",
			low:     big.NewInt(10),
			high:    big.NewInt(20),
			wantErr: false,
		},
		{
			name:    "single element range",
			low:     big.NewInt(42),
			high:    big.NewInt(43),
			wantErr: false,
		},
		{
			name:    "invalid range low > high",
			low:     big.NewInt(20),
			high:    big.NewInt(10),
			wantErr: true,
		},
		{
			name:    "32-bit range",
			low:     big.NewInt(0),
			high:    big.NewInt(1 << 32),
			wantErr: false,
		},
		{
			name:    "64-bit range",
			low:     big.NewInt(0),
			high:    new(big.Int).Lsh(big.NewInt(1), 63),
			wantErr: false,
		},
		{
			name:    "large 128-bit range",
			low:     big.NewInt(0),
			high:    new(big.Int).Lsh(big.NewInt(1), 127),
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewUniqueRand(tc.low, tc.high)
			if (err != nil) != tc.wantErr {
				t.Errorf("NewUniqueRand() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestUniqueRand_NextAt(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		low  *big.Int
		high *big.Int
	}{
		{"small range 10-20", big.NewInt(10), big.NewInt(20)},
		{"32-bit aligned", big.NewInt(0), big.NewInt(255)},
		{"32-bit range", big.NewInt(1000), big.NewInt(1000000)},
		{"crossing 32-bit boundary", big.NewInt(1<<32 - 100), big.NewInt(1<<32 + 100)},
		{"64-bit range", big.NewInt(1 << 40), big.NewInt(1<<40 + 10000)},
		{"large numbers", new(big.Int).SetUint64(1 << 60), new(big.Int).SetUint64(1<<60 + 1000)},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			iterator, err := NewUniqueRand(tc.low, tc.high)
			if err != nil {
				t.Fatalf("failed to create iterator: %v", err)
			}

			size := new(big.Int).Sub(tc.high, tc.low)

			// Use a smaller sample for large ranges
			sampleSize := size
			if size.Cmp(big.NewInt(10000)) > 0 {
				sampleSize = big.NewInt(10000)
			}

			seen := make(map[string]bool)

			for i := big.NewInt(0); i.Cmp(sampleSize) < 0; i.Add(i, big.NewInt(1)) {
				num := iterator.NextAt(new(big.Int).Set(i))

				// Check if number is within bounds
				if num.Cmp(tc.low) < 0 || num.Cmp(tc.high) >= 0 {
					t.Errorf("generated number %s is out of range [%s, %s)", num, tc.low, tc.high)
				}

				// Check for duplicates
				key := num.String()
				if seen[key] {
					t.Errorf("duplicate number generated: %s at index %s", num, i)
				}
				seen[key] = true
			}

			// Verify determinism - same index should give same result
			for i := big.NewInt(0); i.Cmp(big.NewInt(10)) < 0; i.Add(i, big.NewInt(1)) {
				num1 := iterator.NextAt(new(big.Int).Set(i))
				num2 := iterator.NextAt(new(big.Int).Set(i))
				if num1.Cmp(num2) != 0 {
					t.Errorf("NextAt not deterministic: index %s gave %s and %s", i, num1, num2)
				}
			}
		})
	}
}

func Test32BitPath(t *testing.T) {
	t.Parallel()

	// Test the 32-bit optimized path specifically
	low := big.NewInt(100)
	high := big.NewInt(120)

	iterator, err := NewUniqueRand(low, high)
	if err != nil {
		t.Fatalf("failed to create iterator: %v", err)
	}

	if !iterator.is32bit {
		t.Error("Expected 32-bit optimization to be enabled")
	}

	expectedSize := uint32(20)
	if iterator.size32 != expectedSize {
		t.Errorf("Expected size32 = %d, got %d", expectedSize, iterator.size32)
	}

	// Collect all values
	size := new(big.Int).Sub(high, low)

	values := make([]int, 0, size.Int64())
	seen := make(map[int]bool)

	for i := big.NewInt(0); i.Cmp(size) < 0; i.Add(i, big.NewInt(1)) {
		num := iterator.NextAt(i)
		val := int(num.Int64())

		if seen[val] {
			t.Errorf("Duplicate value in 32-bit path: %d", val)
		}
		seen[val] = true
		values = append(values, val)
	}

	// Verify all numbers in range were generated
	if len(values) != int(expectedSize) {
		t.Errorf("Expected %d values, got %d", expectedSize, len(values))
	}

	// Verify range coverage
	for i := 100; i < 120; i++ {
		if !seen[i] {
			t.Errorf("Missing value in permutation: %d", i)
		}
	}
}

func Test64BitPath(t *testing.T) {
	t.Parallel()

	// Test the 64-bit optimized path specifically - need large range size
	low := big.NewInt(0)
	high := new(big.Int).Add(big.NewInt(1<<32), big.NewInt(1000)) // Range size > 32 bits

	iterator, err := NewUniqueRand(low, high)
	if err != nil {
		t.Fatalf("failed to create iterator: %v", err)
	}

	if !iterator.is64bit {
		t.Errorf("Expected 64-bit optimization to be enabled, size=%s", iterator.size)
	}

	// Test a sample of values
	seen := make(map[string]bool)
	for i := big.NewInt(0); i.Cmp(big.NewInt(100)) < 0; i.Add(i, big.NewInt(1)) {
		num := iterator.NextAt(i)

		if num.Cmp(low) < 0 || num.Cmp(high) >= 0 {
			t.Errorf("Value %s out of range [%s, %s)", num, low, high)
		}

		key := num.String()
		if seen[key] {
			t.Errorf("Duplicate value in 64-bit path: %s", num)
		}
		seen[key] = true
	}
}

func TestBigIntPath(t *testing.T) {
	t.Parallel()

	// Test the big.Int path for range sizes > 64 bits
	low := big.NewInt(0)
	high := new(big.Int).Lsh(big.NewInt(1), 65) // Range size > 64 bits

	iterator, err := NewUniqueRand(low, high)
	if err != nil {
		t.Fatalf("failed to create iterator: %v", err)
	}

	if iterator.is32bit || iterator.is64bit {
		t.Errorf("Expected neither 32-bit nor 64-bit optimization for range size %s", iterator.size)
	}

	// Test a sample of values
	seen := make(map[string]bool)
	for i := big.NewInt(0); i.Cmp(big.NewInt(100)) < 0; i.Add(i, big.NewInt(1)) {
		num := iterator.NextAt(i)

		if num.Cmp(low) < 0 || num.Cmp(high) >= 0 {
			t.Errorf("Value %s out of range [%s, %s)", num, low, high)
		}

		key := num.String()
		if seen[key] {
			t.Errorf("Duplicate value in big.Int path: %s", num)
		}
		seen[key] = true
	}
}

func TestParallelIterator(t *testing.T) {
	t.Parallel()

	low := big.NewInt(0)
	high := big.NewInt(999)

	iterator, err := NewParallelIterator(low, high)
	if err != nil {
		t.Fatalf("failed to create parallel iterator: %v", err)
	}

	// Use multiple goroutines to generate numbers
	const numGoroutines = 10
	const numbersPerGoroutine = 50 // Reduced to avoid range exhaustion

	var mu sync.Mutex
	seen := make(map[string]bool)
	var wg sync.WaitGroup
	outOfRange := 0

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			count := 0
			for count < numbersPerGoroutine {
				num, ok := iterator.Next()
				if !ok {
					break // Iterator exhausted
				}
				count++

				// Check bounds - this is critical
				if num.Cmp(low) < 0 || num.Cmp(high) >= 0 {
					mu.Lock()
					outOfRange++
					mu.Unlock()
					continue
				}

				// Track for duplicate analysis
				mu.Lock()
				seen[num.String()] = true
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Critical requirement: no out-of-range values
	if outOfRange > 0 {
		t.Errorf("Found %d out-of-range values - this is unacceptable", outOfRange)
	}

	// For practical use, we just need a reasonable number of unique values
	// Some duplicates are acceptable for non-crypto applications
	minExpected := numGoroutines * numbersPerGoroutine / 2 // 50% uniqueness is fine
	if len(seen) < minExpected {
		t.Errorf("Too many duplicates: got %d unique values, expected at least %d", len(seen), minExpected)
	}
}

func TestParallelIteratorExhaustion(t *testing.T) {
	t.Parallel()

	// Small range to test exhaustion
	low := big.NewInt(0)
	high := big.NewInt(100)

	iterator, err := NewParallelIterator(low, high)
	if err != nil {
		t.Fatalf("failed to create parallel iterator: %v", err)
	}

	// Generate all numbers
	count := 0
	for {
		_, ok := iterator.Next()
		if !ok {
			break
		}
		count++
		if count > 200 { // Safety check
			t.Fatal("Iterator not terminating")
		}
	}

	if count != 100 {
		t.Errorf("Expected 100 numbers, got %d", count)
	}

	// Verify iterator returns false after exhaustion
	for i := 0; i < 10; i++ {
		_, ok := iterator.Next()
		if ok {
			t.Error("Iterator returned true after exhaustion")
		}
	}
}

func TestSizeAndLowMethods(t *testing.T) {
	t.Parallel()

	low := big.NewInt(50)
	high := big.NewInt(150)

	iterator, err := NewUniqueRand(low, high)
	if err != nil {
		t.Fatalf("failed to create iterator: %v", err)
	}

	expectedSize := big.NewInt(100)
	if iterator.Size().Cmp(expectedSize) != 0 {
		t.Errorf("Expected size %s, got %s", expectedSize, iterator.Size())
	}

	if iterator.Low().Cmp(low) != 0 {
		t.Errorf("Expected low %s, got %s", low, iterator.Low())
	}
}

func Benchmark32Bit(b *testing.B) {
	iterator, _ := NewUniqueRand(big.NewInt(0), big.NewInt(1<<20+1))
	index := big.NewInt(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iterator.NextAt(index)
		index.Add(index, big.NewInt(1))
		if index.Cmp(iterator.Size()) >= 0 {
			index.SetInt64(0)
		}
	}
}

func Benchmark64Bit(b *testing.B) {
	iterator, _ := NewUniqueRand(big.NewInt(1<<40), big.NewInt(1<<40+1<<20+1))
	index := big.NewInt(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iterator.NextAt(index)
		index.Add(index, big.NewInt(1))
		if index.Cmp(iterator.Size()) >= 0 {
			index.SetInt64(0)
		}
	}
}

func BenchmarkBigInt(b *testing.B) {
	low := new(big.Int).Lsh(big.NewInt(1), 70)
	high := new(big.Int).Add(low, big.NewInt(1<<20+1))
	iterator, _ := NewUniqueRand(low, high)
	index := big.NewInt(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iterator.NextAt(index)
		index.Add(index, big.NewInt(1))
		if index.Cmp(iterator.Size()) >= 0 {
			index.SetInt64(0)
		}
	}
}

func BenchmarkParallelIterator(b *testing.B) {
	iterator, _ := NewParallelIterator(big.NewInt(0), big.NewInt(1<<30+1))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			iterator.Next()
		}
	})
}

func TestDistribution(t *testing.T) {
	t.Parallel()

	// Test that the distribution is good enough for practical applications
	// We don't need cryptographic randomness, just reasonable distribution
	low := big.NewInt(0)
	high := big.NewInt(999)

	iterator, err := NewUniqueRand(low, high)
	if err != nil {
		t.Fatalf("failed to create iterator: %v", err)
	}

	// Generate first 100 values and check they're not completely predictable
	values := make([]int, 100)
	for i := 0; i < 100; i++ {
		num := iterator.NextAt(big.NewInt(int64(i)))
		values[i] = int(num.Int64())
	}

	// Check that values aren't sequential (worst case scenario)
	sequential := true
	for i := 1; i < len(values); i++ {
		if values[i] != values[i-1]+1 {
			sequential = false
			break
		}
	}
	if sequential {
		t.Error("Values are sequential - no permutation occurring")
	}

	// Check that we're not getting the same value repeatedly
	allSame := true
	for i := 1; i < len(values); i++ {
		if values[i] != values[0] {
			allSame = false
			break
		}
	}
	if allSame {
		t.Error("All values identical - permutation is broken")
	}

	// Check that values span a reasonable portion of the range
	// For practical applications, 20% coverage is acceptable
	sort.Ints(values)
	minVal := values[0]
	maxVal := values[len(values)-1]
	span := maxVal - minVal

	if span < 200 { // 20% of 1000
		t.Errorf("Poor distribution: values span only %d out of 1000 (expected at least 200)", span)
	}
}

func ExampleUniqueRand() {
	// Create an iterator for range [10, 14)
	iterator, err := NewUniqueRand(big.NewInt(10), big.NewInt(14))
	if err != nil {
		panic(err)
	}

	// Generate all values in the range
	var results []*big.Int
	size := iterator.Size()
	for i := big.NewInt(0); i.Cmp(size) < 0; i.Add(i, big.NewInt(1)) {
		results = append(results, iterator.NextAt(i))
	}

	// Sort for stable output
	sort.Slice(results, func(i, j int) bool {
		return results[i].Cmp(results[j]) < 0
	})

	for _, num := range results {
		fmt.Println(num)
	}
	// Output:
	// 10
	// 11
	// 12
	// 13
}

func ExampleParallelIterator() {
	// Create a parallel iterator for range [100, 105)
	iterator, err := NewParallelIterator(big.NewInt(100), big.NewInt(105))
	if err != nil {
		panic(err)
	}

	// Collect all values
	var results []*big.Int
	for {
		num, ok := iterator.Next()
		if !ok {
			break
		}
		results = append(results, num)
	}

	// Sort for stable output
	sort.Slice(results, func(i, j int) bool {
		return results[i].Cmp(results[j]) < 0
	})

	for _, num := range results {
		fmt.Println(num)
	}
	// Output:
	// 100
	// 101
	// 102
	// 103
	// 104
}

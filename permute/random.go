package permute

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// RandomParallelIterator wraps ParallelIterator to provide non-deterministic
// sequences by applying random offsets to both the input range and output values.
// This breaks the deterministic property while maintaining the performance
// characteristics of the underlying permutation algorithm.
//
// Each RandomParallelIterator instance will produce a different sequence,
// even when created with identical parameters, making it suitable for
// applications that require unpredictable iteration orders.
//
// Example usage:
//
//	// Each iterator will produce different sequences
//	iter1, _ := permute.NewRandomParallelIterator(big.NewInt(0), big.NewInt(1000))
//	iter2, _ := permute.NewRandomParallelIterator(big.NewInt(0), big.NewInt(1000))
//
//	// iter1 and iter2 will visit the same numbers but in different orders
//	for {
//	    num, ok := iter1.Next()
//	    if !ok { break }
//	    // Process num - will be in range [0, 1000) but unpredictable order
//	}
type RandomParallelIterator struct {
	iter         *ParallelIterator
	rangeOffset  *big.Int // Random offset applied to input range
	outputOffset *big.Int // Random offset applied to output values
	originalLow  *big.Int // Original low bound for output mapping
	originalHigh *big.Int // Original high bound for output mapping
	size         *big.Int // Size of the original range
}

// NewRandomParallelIterator creates a new non-deterministic parallel iterator
// for the range [low, high). The iterator will visit each number in the range
// exactly once, but in a randomized order that differs between instances.
//
// The randomization works by:
// 1. Generating a random offset for the internal permutation range
// 2. Generating a random offset for mapping outputs back to the target range
// 3. Using crypto/rand for cryptographically secure randomness
//
// Parameters:
//   - low: The lower bound of the range (inclusive)
//   - high: The upper bound of the range (exclusive)
//
// Returns:
//   - A new RandomParallelIterator instance
//   - An error if the range is invalid or random generation fails
//
// Example:
//
//	iter, err := NewRandomParallelIterator(big.NewInt(100), big.NewInt(201))
//	if err != nil {
//	    return err
//	}
//
//	// Use like a regular ParallelIterator
//	for {
//	    num, ok := iter.Next()
//	    if !ok { break }
//	    // num will be in [100, 201) but in randomized order
//	}
func NewRandomParallelIterator(low, high *big.Int) (*RandomParallelIterator, error) {
	if low.Cmp(high) > 0 {
		return nil, fmt.Errorf("low bound %s cannot be greater than high bound %s", low.String(), high.String())
	}

	// Calculate original range size
	size := new(big.Int).Sub(high, low)

	// Generate random offsets using crypto/rand for better randomness
	maxOffset := new(big.Int).Lsh(big.NewInt(1), 32) // Use 32-bit random offsets

	rangeOffset, err := rand.Int(rand.Reader, maxOffset)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random range offset: %w", err)
	}

	outputOffset, err := rand.Int(rand.Reader, maxOffset)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random output offset: %w", err)
	}

	// Create shifted range for internal permutation
	// We shift both bounds by the same offset to maintain range size
	shiftedLow := new(big.Int).Add(low, rangeOffset)
	shiftedHigh := new(big.Int).Add(high, rangeOffset)

	// Create the underlying ParallelIterator with shifted range
	iter, err := NewParallelIterator(shiftedLow, shiftedHigh)
	if err != nil {
		return nil, fmt.Errorf("failed to create underlying iterator: %w", err)
	}

	return &RandomParallelIterator{
		iter:         iter,
		rangeOffset:  rangeOffset,
		outputOffset: outputOffset,
		originalLow:  new(big.Int).Set(low),
		originalHigh: new(big.Int).Set(high),
		size:         size,
	}, nil
}

// Next returns the next unique number in the randomized sequence.
// This method is thread-safe and uses atomic operations to ensure
// that each call returns a unique value, even when called concurrently.
//
// The returned numbers are guaranteed to be within the original range
// [low, high] specified when creating the iterator, and each number
// will be returned exactly once.
//
// Returns:
//   - The next number in the randomized sequence
//   - false when all numbers have been generated
//
// Example:
//
//	for {
//	    num, ok := iter.Next()
//	    if !ok {
//	        break  // All numbers generated
//	    }
//	    // Process num - guaranteed to be in original range
//	}
func (ri *RandomParallelIterator) Next() (*big.Int, bool) {
	// Get next number from underlying iterator (in shifted range)
	shiftedNum, ok := ri.iter.Next()
	if !ok {
		return nil, false
	}

	// Remove the range offset to get back to original coordinate space
	originalNum := new(big.Int).Sub(shiftedNum, ri.rangeOffset)

	// Apply output offset for additional randomization
	randomizedNum := new(big.Int).Add(originalNum, ri.outputOffset)

	// Map back to original range using modular arithmetic
	// This ensures the result is always in [originalLow, originalHigh)
	randomizedNum.Mod(randomizedNum, ri.size)
	result := new(big.Int).Add(randomizedNum, ri.originalLow)

	return result, true
}

// Size returns the total number of elements in the range.
// This is useful for determining when iteration is complete.
func (ri *RandomParallelIterator) Size() *big.Int {
	return new(big.Int).Set(ri.size)
}

// Low returns the lower bound of the original range.
func (ri *RandomParallelIterator) Low() *big.Int {
	return new(big.Int).Set(ri.originalLow)
}

// High returns the upper bound of the original range.
func (ri *RandomParallelIterator) High() *big.Int {
	return new(big.Int).Set(ri.originalHigh)
}

// NewRandomUniqueRand creates a non-deterministic UniqueRand iterator
// that provides the same randomization benefits for sequential access
// via NextAt(). Unlike RandomParallelIterator which is for concurrent
// access with Next(), this provides randomized sequential access.
//
// Parameters:
//   - low: The lower bound of the range (inclusive)
//   - high: The upper bound of the range (exclusive)
//
// Returns:
//   - A randomized UniqueRand instance
//   - An error if the range is invalid or random generation fails
//
// Example:
//
//	iter, err := NewRandomUniqueRand(big.NewInt(0), big.NewInt(1000))
//	if err != nil {
//	    return err
//	}
//
//	// Sequential access with randomized order
//	for i := big.NewInt(0); i.Cmp(iter.Size()) < 0; i.Add(i, big.NewInt(1)) {
//	    num := iter.NextAt(i)  // Randomized but deterministic per index
//	    // Process num
//	}
func NewRandomUniqueRand(low, high *big.Int) (*RandomUniqueRand, error) {
	if low.Cmp(high) > 0 {
		return nil, fmt.Errorf("low bound %s cannot be greater than high bound %s", low.String(), high.String())
	}

	// Calculate original range size
	size := new(big.Int).Sub(high, low)

	// Generate random offsets
	maxOffset := new(big.Int).Lsh(big.NewInt(1), 32)

	rangeOffset, err := rand.Int(rand.Reader, maxOffset)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random range offset: %w", err)
	}

	outputOffset, err := rand.Int(rand.Reader, maxOffset)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random output offset: %w", err)
	}

	// Create shifted range for internal permutation
	shiftedLow := new(big.Int).Add(low, rangeOffset)
	shiftedHigh := new(big.Int).Add(high, rangeOffset)

	// Create the underlying UniqueRand with shifted range
	iter, err := NewUniqueRand(shiftedLow, shiftedHigh)
	if err != nil {
		return nil, fmt.Errorf("failed to create underlying iterator: %w", err)
	}

	return &RandomUniqueRand{
		iter:         iter,
		rangeOffset:  rangeOffset,
		outputOffset: outputOffset,
		originalLow:  new(big.Int).Set(low),
		originalHigh: new(big.Int).Set(high),
		size:         size,
	}, nil
}

// RandomUniqueRand provides non-deterministic sequential access to
// a permuted range using NextAt() with random offsets applied.
type RandomUniqueRand struct {
	iter         *UniqueRand
	rangeOffset  *big.Int
	outputOffset *big.Int
	originalLow  *big.Int
	originalHigh *big.Int
	size         *big.Int
}

// NextAt returns the randomized permuted value at a specific index.
// Unlike the deterministic UniqueRand.NextAt(), this applies random
// offsets to make the sequence unpredictable between instances.
//
// The index must be in the range [0, size), where size = high - low.
//
// Example:
//
//	iter, _ := NewRandomUniqueRand(big.NewInt(100), big.NewInt(200))
//
//	// Same index will give different results in different instances
//	num := iter.NextAt(big.NewInt(0))  // Randomized first element
func (ru *RandomUniqueRand) NextAt(index *big.Int) *big.Int {
	// Get value from underlying iterator (in shifted range)
	shiftedNum := ru.iter.NextAt(index)

	// Remove the range offset to get back to original coordinate space
	originalNum := new(big.Int).Sub(shiftedNum, ru.rangeOffset)

	// Apply output offset for additional randomization
	randomizedNum := new(big.Int).Add(originalNum, ru.outputOffset)

	// Map back to original range using modular arithmetic
	randomizedNum.Mod(randomizedNum, ru.size)
	result := new(big.Int).Add(randomizedNum, ru.originalLow)

	return result
}

// Size returns the total number of elements in the range.
func (ru *RandomUniqueRand) Size() *big.Int {
	return new(big.Int).Set(ru.size)
}

// Low returns the lower bound of the original range.
func (ru *RandomUniqueRand) Low() *big.Int {
	return new(big.Int).Set(ru.originalLow)
}

// High returns the upper bound of the original range.
func (ru *RandomUniqueRand) High() *big.Int {
	return new(big.Int).Set(ru.originalHigh)
}

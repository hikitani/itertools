package itertools

import (
	"context"
	"slices"
	"time"

	"golang.org/x/exp/constraints"
)

func OrderedAcc[T constraints.Ordered](acc, v T) T { return acc + v }

// From method receives a generator and iterates over its values,
// calling the Yield function for each value.
func (y Yield[T]) From(gen generator[T]) {
	for v := range gen {
		y(v)
	}
}

// Yield is a type that represents a function that can yield a value.
type Yield[T any] func(v T)

// generator is a type that represents a channel of values.
type generator[T any] chan T

// Close tries to close the generator channel.
func (g generator[T]) Close() {
	defer func() { recover() }()
	close(g)
}

// send tries send a value to the generator channel.
func (g generator[T]) send(v T) {
	defer func() { recover() }()
	g <- v
}

// NewGen creates a new generator by executing the provided function in a separate goroutine.
// The function takes a Yield function as an argument to send values to the generator.
// The generator is closed after the function execution is completed.
func NewGen[T any](f func(yield Yield[T])) generator[T] {
	gen := make(generator[T], 100)

	go func() {
		f(gen.send)
		gen.Close()
	}()

	return gen
}

// Next returns next value from generator.
//
//	v, ok := g.Next()
//	if !ok {
//		fmt.Println("all values generated")
//		return
//	}
//	fmt.Println(v)
func (g generator[T]) Next() (T, bool) {
	v, ok := <-g
	return v, ok
}

// WithDuration returns new generator that receives
// values from parent generator until it passes specified duration.
//
//	for v := range g.WithDuration(10*time.Second) {
//		// iterating
//	}
func (g generator[T]) WithDuration(d time.Duration) generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		ctx, cancel := context.WithTimeout(context.Background(), d)
		defer cancel()
		yield.From(g.WithContext(ctx))
	})
}

// WithContext returns new generator that receives
// values from parent generator until it ctx is done.
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//	for v := range g.WithContext(ctx) {
//		// iterating
//	}
func (g generator[T]) WithContext(ctx context.Context) generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-g:
				if !ok {
					return
				}

				yield(v)
			}
		}
	})
}

// Max returns new generator that receives n
// values from parent generator.
func (g generator[T]) Max(n int) generator[T] {
	return NewGen(func(yield Yield[T]) {
		for i := 0; i < n; i++ {
			v, ok := g.Next()
			if !ok {
				break
			}

			yield(v)
		}
	})
}

// Cycle returns new generator that repeats
// indenfinitely values from parent generator.
//
//	g := itertools.NewGen[int](func(yield itertools.Yield[int]) {
//		yield(1)
//		yield(2)
//		yield(3)
//	})
//
//	for v := range g.Max(5) {
//		fmt.Println(v)
//	}
//
//	// Output:
//	// 1
//	// 2
//	// 3
//	// 1
//	// 2
func (g generator[T]) Cycle() generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		var saved []T
		for v := range g {
			yield(v)
			saved = append(saved, v)
		}
		if len(saved) == 0 {
			return
		}

		for {
			for _, v := range saved {
				yield(v)
			}
		}
	})
}

// Nth returns new generator that receives
// each nth values from parent generator.
//
//	g := itertools.NewGen[int](func(yield itertools.Yield[int]) {
//		yield(1)
//		yield(2)
//		yield(3)
//		yield(4)
//		yield(5)
//		yield(6)
//	})
//
//	for v := range g.Nth(2) {
//		fmt.Println(v)
//	}
//
//	// Output:
//	// 2
//	// 4
//	// 6
func (g generator[T]) Nth(n int) generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		for i := 0; ; i++ {
			v, ok := g.Next()
			if !ok {
				return
			}

			if i%n != 0 {
				continue
			}

			yield(v)
		}
	})
}

// Where returns new generator that receives
// values from parent generator for which predicate is true.
//
//	g := itertools.NewGen[int](func(yield itertools.Yield[int]) {
//		yield(1)
//		yield(2)
//		yield(3)
//		yield(4)
//		yield(5)
//		yield(6)
//	})
//
//	for v := range g.Where(func(v int) bool { return v <= 2 }) {
//		fmt.Println(v)
//	}
//
//	// Output:
//	// 1
//	// 2
func (g generator[T]) Where(predicate func(v T) bool) generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		for v := range g {
			if predicate(v) {
				yield(v)
			}
		}
	})
}

// Where returns new generator that receives
// values from parent generator while predicate is true.
//
//	g := itertools.NewGen[int](func(yield itertools.Yield[int]) {
//		yield(1)
//		yield(2)
//		yield(3)
//		yield(2)
//		yield(1)
//	})
//
//	for v := range g.TakeWhile(func(v int) bool { return v <= 2 }) {
//		fmt.Println(v)
//	}
//
//	// Output:
//	// 1
//	// 2
func (g generator[T]) TakeWhile(predicate func(v T) bool) generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		for v := range g {
			if !predicate(v) {
				return
			}

			yield(v)
		}
	})
}

// Where returns new generator that drops
// values from parent generator while predicate is true.
//
//	g := itertools.NewGen[int](func(yield itertools.Yield[int]) {
//		yield(1)
//		yield(2)
//		yield(3)
//		yield(2)
//		yield(1)
//	})
//
//	for v := range g.TakeWhile(func(v int) bool { return v <= 2 }) {
//		fmt.Println(v)
//	}
//
//	// Output:
//	// 3
//	// 2
//	// 1
func (g generator[T]) DropWhile(predicate func(v T) bool) generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		for v := range g {
			if !predicate(v) {
				break
			}

			yield(v)
		}
	})
}

// Tee returns n independent generators from parent generator.
func (g generator[T]) Tee(n int) []generator[T] {
	var gens []generator[T]
	var deques []*queue[T]
	for i := 0; i < n; i++ {
		deques = append(deques, new(queue[T]))
	}

	for i := 0; i < n; i++ {
		localDeque := deques[i]
		gens = append(gens, NewGen[T](func(yield Yield[T]) {
			for {
				if localDeque.IsEmpty() {
					v, ok := g.Next()
					if !ok {
						return
					}
					for _, d := range deques {
						d.Append(v)
					}
				}

				yield(localDeque.PopLeft())
			}
		}))
	}
	return gens
}

// Prepend adds a value to the beginning of a generator.
//
//	g := itertools.NewGen[int](func(yield itertools.Yield[int]) {
//		yield(2)
//		yield(3)
//	})
//
//	for v := range g.Prepend(1) {
//		fmt.Println(v)
//	}
//
//	// Output:
//	// 1
//	// 2
//	// 3
func (g generator[T]) Prepend(v T) generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		yield(v)
		yield.From(g)
	})
}

// Append adds a value to the end of a generator.
//
//	g := itertools.NewGen[int](func(yield itertools.Yield[int]) {
//		yield(1)
//		yield(2)
//	})
//
//	for v := range g.Append(3) {
//		fmt.Println(v)
//	}
//
//	// Output:
//	// 1
//	// 2
//	// 3
func (g generator[T]) Append(v T) generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		yield.From(g)
		yield(v)
	})
}

// Accumulate applies a function to each value of the generator and
// yields the accumulated result from function.
//
//	g := itertools.NewGen[int](func(yield itertools.Yield[int]) {
//		yield(1)
//		yield(2)
//		yield(3)
//	})
//
//	for v := range g.Accumulate(func(acc, cur int) int { return acc + cur }, 0) {
//		fmt.Println(v)
//	}
//
//	// Output:
//	// 1
//	// 3
//	// 6
func (g generator[T]) Accumulate(f func(acc, cur T) T, initial T) generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		acc := initial
		for v := range g {
			acc = f(acc, v)
			yield(acc)
		}
	})
}

// Skip returns a new generator that skips the first n values of the original generator.
//
//	g := itertools.NewGen[int](func(yield itertools.Yield[int]) {
//		yield(1)
//		yield(2)
//		yield(3)
//		yield(4)
//	})
//
//	for v := range g.Skip(2) {
//		fmt.Println(v)
//	}
//
//	// Output:
//	// 3
//	// 4
func (g generator[T]) Skip(n int) generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		for i := 0; i < n; i++ {
			_, ok := g.Next()
			if !ok {
				return
			}
		}

		yield.From(g)
	})
}

// Unique returns a new generator that only yields unique values from the original generator.
func (g generator[T]) Unique() generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		seen := map[any]struct{}{}
		for v := range g {
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			yield(v)
		}
	})
}

// UniqueJustSeen returns a new generator that yields unique values based on a key function, preserving order.
//
//	g := itertools.NewGen[int](func(yield itertools.Yield[int]) {
//		yield(1)
//		yield(1)
//		yield(2)
//		yield(2)
//		yield(1)
//	})
//
//	for v := range g.UniqueJustSeen(func(v int) any { return v }) {
//		fmt.Println(v)
//	}
//
//	// Output:
//	// 1
//	// 2
//	// 1
func (g generator[T]) UniqueJustSeen(key func(v T) any) generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		v, ok := g.Next()
		if !ok {
			return
		}

		yield(v)
		var lastSeen any = v
		for v := range g {
			if any(v) != lastSeen {
				lastSeen = v
				yield(v)
			}
		}
	})
}

// List returns a slice of values from the generator, up to a maximum of max values.
func (g generator[T]) List(max int) (s []T) {
	for v := range g.Max(max) {
		s = append(s, v)
	}

	return
}

// Before takes multiple generators and yields their values before main generator.
func (g generator[T]) Before(gens ...generator[T]) generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		for _, gen := range gens {
			yield.From(gen)
		}

		yield.From(g)
	})
}

// After takes multiple generators and yields their values after main generator.
func (g generator[T]) After(gens ...generator[T]) generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		yield.From(g)

		for _, gen := range gens {
			yield.From(gen)
		}
	})
}

// Count generates a sequence of values starting from 'start' with a step of 'step'.
func Count[T constraints.Ordered](start, step T) generator[T] {
	return NewGen(func(yield Yield[T]) {
		for i := start; ; i += step {
			yield(i)
		}
	})
}

// CountMap generates a sequence of values starting from a given value and applying a step function.
//
//	for v := range itertools.CountMap(10,
//		func(v int) int {
//			if v%2 == 0 {
//				return v / 2
//			}
//
//			return 3*v + 1
//		}).Max(5) {
//		fmt.Println(v)
//	}
//
//	// Output:
//	// 10
//	// 5
//	// 16
//	// 8
//	// 4
func CountMap[T any](start T, stepFunc func(v T) T) generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		for i := start; ; i = stepFunc(i) {
			yield(i)
		}
	})
}

// Repeat generates a sequence of values repeating 'v' 'times' number of times.
func Repeat[T any](v T, times int) generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		for i := 0; i < times; i++ {
			yield(v)
		}
	})
}

// Chain chains multiple generators together into a single generator.
func Chain[T any](gens ...generator[T]) generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		for _, g := range gens {
			yield.From(g)
		}
	})
}

// Join takes multiple generators and combines their values into slices of type T.
func Join[T any](gens ...generator[T]) generator[[]T] {
	return NewGen[[]T](func(yield Yield[[]T]) {
		for {
			vals := make([]T, len(gens))
			for i, g := range gens {
				v, ok := g.Next()
				if !ok {
					return
				}
				vals[i] = v
			}

			yield(vals)
		}
	})
}

// Map applies a conversion function to each value produced by the generator.
func Map[T1, T2 any](gen generator[T1], conv func(v T1) T2) generator[T2] {
	return NewGen[T2](func(yield Yield[T2]) {
		for v := range gen {
			yield(conv(v))
		}
	})
}

// SlidingWindow creates a generator that returns sliding windows of a given size and offset.
//
//	g := itertools.NewGen[int](func(yield itertools.Yield[int]) {
//		yield(1)
//		yield(2)
//		yield(3)
//		yield(4)
//		yield(5)
//	})
//
//	for v := range itertools.SlidingWindow(g, 3, 1) {
//		fmt.Println(v)
//	}
//
//	// Output:
//	// [1 2 3]
//	// [2 3 4]
//	// [3 4 5]
func SlidingWindow[T any](gen generator[T], size, offset int) generator[[]T] {
	return NewGen[[]T](func(yield Yield[[]T]) {
		slide := make([]T, 0, size)
		for i := 0; i < size; i++ {
			v, ok := gen.Next()
			if !ok {
				yield(slide)
				return
			}

			slide = append(slide, v)
		}

		yield(slices.Clone(slide))
		for v := range gen {
			slide = slide[offset:]
			slide = append(slide, v)
			yield(slices.Clone(slide))
		}
	})
}

// Convolve applies a convolution operation to a generator using a given kernel.
//
//   - kernel = [0.25, 0.25, 0.25, 0.25] -> moving average
//   - kernel = [1/2, 0, -1/2] -> 1st derivative estimate
//   - kernel = [1, -2, 1] -> 2nd derivative estimate
func Convolve[T constraints.Float](gen generator[T], kernel []T) generator[T] {
	return NewGen[T](func(yield Yield[T]) {
		kernel := slices.Clone(kernel)
		slices.Reverse(kernel)
		paddedData := gen.Before(Repeat[T](0, len(kernel)-1))

		slider := SlidingWindow[T](paddedData, len(kernel), 1)
		for slide := range slider {
			var sumProd T
			for i := 0; i < len(kernel); i++ {
				sumProd += slide[i] * kernel[i]
			}

			yield(sumProd)
		}
	})
}

package itertools

// GroupElement represents a key-value pair in a group.
type GroupElement[K comparable, V any] struct {
	Key   K
	Value V
}

// Group is a generator that yields groups of type T.
type Group[K comparable, V any] struct {
	generator[GroupElement[K, V]]
}

// AsGen just converts type Group to generator.
func (g Group[K, V]) AsGen() generator[GroupElement[K, V]] {
	return g.generator
}

// GroupBy groups consecutive the values from a generator based on a key function.
// Generally, the generator should generate sorted by the same key function.
//
//	g := itertools.NewGen[string](func(yield Yield[string]) {
//		yield("A")
//		yield("A")
//		yield("B")
//		yield("B")
//		yield("B")
//		yield("A")
//	})
//
//	keySelector := func(v string) string { return v }
//	groupElementMapper := func(v itertools.GroupElement[string, string]) string { return v.Value }
//
//	for group := range itertools.GroupBy(g, keySelector) {
//		groupValues := itertools.Map(group.AsGen(), groupElementMapper)
//		fmt.Println(groupValues.List(10))
//	}
//
//	// Output:
//	// [A A]
//	// [B B B]
//	// [A]
func GroupBy[K comparable, V any](g generator[V], key func(V) K) generator[Group[K, V]] {
	return NewGen[Group[K, V]](func(yield Yield[Group[K, V]]) {
		var (
			lastKey   K
			lastGroup []V
		)

		i := -1
		for v := range g {
			i++
			var curKey K
			if i == 0 {
				lastKey = key(v)
				curKey = lastKey
			} else {
				curKey = key(v)
			}

			if curKey == lastKey {
				lastGroup = append(lastGroup, v)
				continue
			}

			key := lastKey
			group := lastGroup

			lastKey = curKey
			lastGroup = []V{v}

			yield(genGroupSlice(group, key))
		}

		if lastGroup != nil {
			yield(genGroupSlice(lastGroup, lastKey))
		}
	})
}

// genGroupSlice generates a group based on a slice of values and a key.
func genGroupSlice[K comparable, V any](s []V, key K) Group[K, V] {
	return Group[K, V]{NewGen[GroupElement[K, V]](func(yield Yield[GroupElement[K, V]]) {
		for _, v := range s {
			yield(GroupElement[K, V]{
				Key:   key,
				Value: v,
			})
		}
	})}
}

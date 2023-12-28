package itertools

import (
	"math/rand"
	"time"
)

// RandomStringsWithLetters generates random strings using the given letters, with lengths between min and max.
func RandomStringsWithLetters(letters string, min, max int) generator[string] {
	src := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(src)

	return NewGen[string](func(yield Yield[string]) {
		b := make([]byte, max)
		for {
			length := rnd.Intn(max-min) + min
			for i := 0; i < length; i++ {
				b[i] = letters[rand.Intn(len(letters))]
			}
			yield(string(b[:length]))
		}
	})
}

// RandomStrings generates random strings using lowercase and uppercase letters, with lengths between min and max.
func RandomStrings(min, max int) generator[string] {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	return RandomStringsWithLetters(letterBytes, min, max)
}

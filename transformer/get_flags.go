package transformer

// Port of src/TSTransformer/util/getFlags.ts.

// for debugging purposes: extracts flag names from a bitfield

// getFlagsEntry is one entry of the `from` record. A slice keeps the order that Object.entries gives.
type getFlagsEntry[T ~int | ~int32 | ~uint32 | ~int64 | ~uint64] struct {
	name  string
	value T
}

func getFlags[T ~int | ~int32 | ~uint32 | ~int64 | ~uint64](flags T, from []getFlagsEntry[T]) []string {
	results := []string{}
	for _, entry := range from {
		if flags&entry.value != 0 {
			results = append(results, entry.name)
		}
	}
	return results
}

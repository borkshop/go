package main

func WriteNextRandomInt64Vector(entropy []int64) {
	for i := 0; i < len(entropy); i++ {
		state := uint64(entropy[i])
		state *= 6364136223846793005
		state += 1442695040888963407
		state ^= state >> 12
		state ^= state << 25
		state ^= state >> 27
		entropy[i] = int64(state)
	}
}

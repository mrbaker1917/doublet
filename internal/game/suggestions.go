package game

import (
	"crypto/rand"
	"math/big"
)

var mediumDoublets = [][2]string{
	{"cold", "warm"},
	{"hand", "foot"},
	{"head", "tail"},
	{"more", "less"},
	{"dark", "dawn"},
	{"four", "five"},
	{"hate", "love"},
	{"fire", "hide"},
	{"ring", "song"},
	{"swim", "flew"},
	{"wine", "beer"},
	{"work", "play"},
	{"left", "mine"},
	{"hunt", "fish"},
	{"word", "game"},
}

var hardDoublets = [][2]string{
	{"stone", "money"},
	{"witch", "bride"},
	{"black", "white"},
	{"blood", "sweat"},
	{"bread", "toast"},
	{"floor", "glass"},
	{"night", "light"},
	{"grass", "green"},
	{"chain", "break"},
	{"sleep", "dream"},
}

var easyDoublets = [][2]string{
	{"cat", "dog"},
	{"hit", "hot"},
	{"bat", "cat"},
	{"rat", "bat"},
	{"hat", "cat"},
	{"bit", "bat"},
	{"pit", "pat"},
	{"pat", "cat"},
	{"sit", "sat"},
	{"sat", "cat"},
	{"mat", "hat"},
	{"fat", "cat"},
	{"bed", "bad"},
	{"red", "bed"},
	{"fed", "bed"},
	{"led", "bed"},
	{"men", "pen"},
	{"hen", "pen"},
	{"ten", "pen"},
	{"den", "pen"},
	{"big", "bag"},
	{"dig", "dog"},
	{"fog", "dog"},
	{"log", "dog"},
	{"cot", "cat"},
	{"cut", "cat"},
	{"cup", "cap"},
	{"cap", "cat"},
	{"car", "cat"},
	{"can", "cat"},
}

func randIntn(n int) int {
	val, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(val.Int64())
}

func GetSuggestedDoublets() ([2]string, [2]string, [2]string) {
	return easyDoublets[randIntn(len(easyDoublets))],
		mediumDoublets[randIntn(len(mediumDoublets))],
		hardDoublets[randIntn(len(hardDoublets))]
}

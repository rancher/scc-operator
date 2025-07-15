package salt

import (
	"math/rand"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type Generator struct {
	randSrc     *rand.Rand
	saltCharset string
	charsetLen  int
}

func NewSaltGen(timeIn *time.Time, charsetIn *string) *Generator {
	if timeIn == nil {
		now := time.Now()
		timeIn = &now
	}
	randSrc := rand.New(rand.NewSource(timeIn.UnixNano()))

	setCharset := charset
	if charsetIn != nil {
		setCharset = *charsetIn
	}

	return &Generator{randSrc: randSrc, saltCharset: setCharset, charsetLen: len(setCharset)}
}

func (s *Generator) GenerateCharacter() uint8 {
	randIndex := s.randSrc.Intn(s.charsetLen)
	return s.saltCharset[randIndex]
}

func (s *Generator) GenerateSalt() string {
	salt := make([]byte, 8)
	for i := range salt {
		salt[i] = s.GenerateCharacter()
	}

	return string(salt)
}

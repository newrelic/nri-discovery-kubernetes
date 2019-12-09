package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Contains_Return_True(t *testing.T) {

	//given
	set := []string{
		"a",
		"abcd",
		"abcdef",
	}
	term := "abcd"

	//when
	contains := Contains(set, term)

	//then
	assert.True(t, contains)
}

func Test_Contains_Return_False(t *testing.T) {

	//given
	set := []string{
		"a",
		"abcd",
		"abcdef",
	}
	term := "zxyw"

	//when
	contains := Contains(set, term)

	//then
	assert.False(t, contains)
}

func Test_Contains_Return_False_IfEmptySet(t *testing.T) {

	//given
	var set []string
	term := "string"

	//when
	contains := Contains(set, term)

	//then
	assert.False(t, contains)
}

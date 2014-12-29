package main

import (
	"github.com/kr/pretty"
	"testing"
)

func TestRecipe(t *testing.T) {
	br := NewRecipe()
	pretty.Printf("%# v\n\n", br.advertisers)
}

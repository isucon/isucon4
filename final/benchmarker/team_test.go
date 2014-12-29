package main

import (
	"fmt"
	"testing"
)

func TestGetTeam(t *testing.T) {
	team := GetTeamByApiKey("1--neodu7-335h-82bb96096cb6f76d38073ef54a13d9be3b08b9ba")
	fmt.Printf("%# v\n", team)
}

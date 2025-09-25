/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"

	"github.com/ashupednekar/compose/cmd"
	"github.com/ashupednekar/compose/pkg"
)

func main() {
	if _, err := pkg.LoadSettings(); err != nil{
		fmt.Println(err)
		return
	}
	cmd.Execute()
}

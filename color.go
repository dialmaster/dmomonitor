package main

import (
	"fmt"
)

var colorRed = "\033[31m"
var colorGreen = "\033[32m"
var colorYellow = "\033[33m"
var colorBlue = "\033[34m"
var colorPurple = "\033[35m"
var colorCyan = "\033[36m"
var colorWhite = "\033[37m"
var colorBrightWhite = "\u001b[37;1m"

func setColor(color string) {
	fmt.Printf(color)
}

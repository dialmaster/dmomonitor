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
var colorBrightMagenta = "\u001b[35;1m"
var colorBrightYellow = "\u001b[33;1m"
var colorBrightCyan = "\u001b[36;1m"
var colorBrightBlue = "\u001b[34;1m"
var colorBrightGreen = "\u001b[32;1m"

func setColor(color string) {
	fmt.Printf(color)
}

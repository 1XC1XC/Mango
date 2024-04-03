package main

import (
    "fmt"
    "os"
)

func main() {
    fmt.Fprintf(os.Stdout, "\033[?25l")
    defer fmt.Fprintf(os.Stdout, "\033[?25h")
    CLI()
    CleanMangoCache(MangoPath)
}

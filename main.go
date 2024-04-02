package main

func main() {
    fmt.Fprintf(os.Stdout, "\033[?25l")
	CLI()
    CleanMangoCache(MangoPath)
    fmt.Fprintf(os.Stdout, "\033[?25h")
}

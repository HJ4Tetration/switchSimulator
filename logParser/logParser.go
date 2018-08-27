package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	filePath := "FD021422KGMlogInfo"
	inFile, _ := os.Open(filePath)
	defer inFile.Close()
	scanner := bufio.NewScanner(inFile)
	scanner.Split(bufio.ScanLines)
	tmp := make([]string, 5)
	i := 0
	for scanner.Scan() {
		tmp[i] = strings.Replace(scanner.Text(), "\\", "", -1)
		i++
	}
	fmt.Printf(tmp[1] + "\n")
}

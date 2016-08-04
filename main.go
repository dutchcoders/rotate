package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"time"
)

func main() {
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Println("Usage: rotate ping 8.8.8.8")
		return
	}

	rows := 0
	cols := 0

	if r, c, err := getSize(); err != nil {
		panic(err)
	} else {
		rows = r
		cols = c
	}

	old, err := MakeRaw(0)
	if err != nil {
		panic(err)
	}

	defer Restore(0, old)

	if fi, err := os.Stdin.Stat(); err != nil {
		panic(err)
	} else if fi.Mode()&os.ModeNamedPipe > 0 {
	}

	a := New(os.Stdout)

	a.Reset().DisableCursor()

	defer a.EnableCursor()

	cmd := exec.Command(flag.Args()[0], flag.Args()[1:]...)

	sout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	serr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for {
			cmd.Process.Signal(<-c)
		}
	}()

	printCh := make(chan string)
	errorCh := make(chan string)
	go func() {
		y := int(0)

		for {
			var s string

			select {
			case s = <-printCh:
				fmt.Fprintf(os.Stdout, "\x1b[9999D\x1b[K")
			case s = <-errorCh:
				fmt.Fprintf(os.Stdout, "\x1b[9999D\x1b[K\x1b[;31m")
			}

			fmt.Fprintf(os.Stdout, "%s\x1b[0m", s)

			d := time.Now().Format("15:04:05")
			fmt.Fprintf(os.Stdout, "\x1b[%dC\x1b[;30;1m%s\x1b[0m", cols-len(s)-len(d), d)

			y++

			if y > rows {
				fmt.Fprintf(os.Stdout, "\x1b[0;0H")
				y = 0
			}

			fmt.Fprintf(os.Stdout, "\x1b[%d;0H\x1b[K>", y)
		}

	}()

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		br := bufio.NewReader(sout)
		for {
			line, _, err := br.ReadLine()
			if err != nil {
				break
			}
			printCh <- string(line)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		br := bufio.NewReader(serr)
		for {
			line, _, err := br.ReadLine()
			if err != nil {
				break
			}
			errorCh <- string(line)
		}
	}()

	cmd.Start()
	cmd.Wait()

	// wait for buffers to finish
	wg.Wait()
}

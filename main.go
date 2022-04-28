package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
	"strings"
)

var (
	command   *exec.Cmd
	startTime time.Time
	modTime   time.Time
	writer    io.Writer = os.Stdout
	done                = make(chan error)
)

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func runBin(bin string, args []string) error {
	cmd := exec.Command(bin, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	startTime = time.Now()

	go io.Copy(writer, stdout)
	go io.Copy(writer, stderr)

	go func() {
		done <- cmd.Wait()
	}()

	command = cmd
	return nil
}

func kill() error {
	if command == nil || command.Process == nil {
		return nil
	}

	err := command.Process.Kill()
	if err != nil {
		return err
	}

	return <-done // allow goroutine to exit
}

func main() {
	bin := ""
	args := []string{}

	if len(os.Args) < 2 {
		fmt.Println("Missing args, require at least one usage: [program] binary arg1 arg2 arg3")
		os.Exit(1)
	}

	for idx, arg := range os.Args {
		if idx == 0 {
			// Skip the first arg as its our program name.
			continue
		} else if idx == 1 {
			bin = arg
		} else {
			args = append(args, arg)
		}
	}

	_, err := os.Stat(bin)
	checkError(err)
	if os.IsNotExist(err) {
		fmt.Printf("Path: %s does not exist\n", bin)
		os.Exit(1)
	}

	log.Println("Initially starting the app...")
	err = runBin(bin, args)
	checkError(err)

	nbusy := 0

	for {
		time.Sleep(500 * time.Millisecond)

		stat, err := os.Stat(bin)
		if err != nil {
			// Not found
			continue
		}

		modTime = stat.ModTime()
		if modTime.After(startTime) {
			log.Println("Reloading the app...")

			// Deliberately ignoring errors here.
			kill()
			command = nil

			// Need sleeping before starting the app or it will somewtimes fail.
			// Need to investigate why exactly, some timing issue.
			// Might be caused by https://github.com/golang/go/issues/22220
			time.Sleep(500 * time.Millisecond)
			log.Println("App killed, starting it again...")

			stat, err = os.Stat(bin)
			if err != nil {
				// Not found
				continue
			}

			err = runBin(bin, args)
			// https://github.com/golang/go/issues/22220
			if err != nil && nbusy < 3 && strings.Contains(err.Error(), "text file busy") {
				log.Println("Text file busy - retry in a bit")
				time.Sleep(100 * time.Millisecond << uint(nbusy))
				nbusy++
				continue
			}
			checkError(err)

			nbusy = 0
			log.Println("Started and up and running...")
		}
	}
}

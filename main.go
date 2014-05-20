package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"time"
)

var (
	command   *exec.Cmd
	startTime time.Time
	modTime   time.Time
	writer    io.Writer = ioutil.Discard
)

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func runBin(bin string, args []string) error {
	command = exec.Command(bin, args...)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return err
	}

	err = command.Start()
	if err != nil {
		return err
	}

	startTime = time.Now()

	go io.Copy(writer, stdout)
	go io.Copy(writer, stderr)
	time.Sleep(250 * time.Millisecond)
	return nil
}

func kill() error {
	command.Process.Release()
	return command.Process.Kill()
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

	err = runBin(bin, args)
	checkError(err)

	for {
		time.Sleep(500 * time.Millisecond)

		stat, err := os.Stat(bin)
		checkError(err)

		modTime = stat.ModTime()
		if modTime.After(startTime) {
			fmt.Println("Reloading the app...")

			err = kill()
			checkError(err)

			err = runBin(bin, args)
			checkError(err)
		}
	}
}

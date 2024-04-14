package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/creack/pty"
	"github.com/draychev/go-toolbox/pkg/logger"
)

var log = logger.NewPretty("lpsh")

const (
	lpDevice  = "/dev/usb/lp0" /* Can we automaticall detect the printer? */
	shCommand = "/bin/sh"
)

func rewriteBackspace(data []byte) []byte {
	out := make([]byte, len(data))
	for _, b := range data {
		switch b {
		case '\b':
			out = append(out, []byte("^H")...)
		default:
			out = append(out, b)
		}
	}
	return out
}

func lpmgr(in chan (interface{}), out chan ([]byte)) {
	// TODO: Runtime configurable option? Discover printers? dunno
	f, err := os.OpenFile(lpDevice, os.O_RDWR, 0755)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error opening %s", lpDevice)
	}

	feed := false
	// ESC/P commands to configure the printer
	configPrinter := [][]byte{
		{27, 'W', '0'}, // turn off double-width
		{27, 'w', '0'}, // turn off double-hight
		// {27, 'M'} // set the font to 12 CPI (ESC M)
		{27, 'g'}, // set the font to 15 CPI (ESC g)
		{27, '0'}, // set the line spacing
	}
	for _, escCommand := range configPrinter {
		f.Write(escCommand)
	}

	f.Write([]byte("\n\n\n\r"))

	const (
		waitAfterKey = 3 * time.Second
		feedIn       = "\x1Bj\xD8\x1Bj\xD8\x1Bj\x6C" // feed it back in
		rollOut      = "\x1BJ\xD8\x1BJ\xD8\x1BJ\x6C" // pop up the paper
	)
	timeout := waitAfterKey
	for {
		select {
		case <-in:
			timeout = waitAfterKey
		case data := <-out:
			if feed {
				f.Write([]byte(feedIn)) // Move down so we can print
				feed = false
			}
			f.Write(rewriteBackspace(data)) // Write the output to the printer
		case <-time.After(timeout):
			timeout = waitAfterKey // was: 200 * time.Millisecond
			if !feed {
				feed = true
				f.Write([]byte(rollOut))
			}
		}
	}
}

func main() {
	winsize := pty.Winsize{Cols: 160, Rows: 24}
	cmd := exec.Command(shCommand)

	cmd.Env = append(os.Environ(), "TERM=lp", fmt.Sprintf("COLUMNS=%d", 180))
	tty, err := pty.StartWithSize(cmd, &winsize)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error starting command %s", shCommand)
	}

	// TODO: This could be done with tcsetattr if we cared enough
	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()

	inch := make(chan (interface{}))
	outch := make(chan ([]byte))
	go lpmgr(inch, outch)

	inbuf := make([]byte, 4096)
	go func() {
		for {
			n, err := os.Stdin.Read(inbuf)
			if err != nil {
				log.Fatal().Err(err).Msg("Error reading from stdin")
			}
			tty.Write(inbuf[:n])
			inch <- nil
		}
	}()

	outbuf := make([]byte, 4096)
	for {
		n, err := tty.Read(outbuf)
		if err != nil {
			log.Fatal().Err(err).Msg("Error reading tty")
		}
		b := make([]byte, n)
		copy(b, outbuf[:n])
		outch <- b
	}
}

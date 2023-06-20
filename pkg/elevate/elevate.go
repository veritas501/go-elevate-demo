package elevate

import (
	"bufio"
	"fmt"
	"github.com/Microsoft/go-winio"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/sys/windows"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	SW_HIDE   int = 0
	SW_NORMAL int = 1
)

func IsAdmin() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

var (
	IsElevateMode   bool
	stdinNamedPipe  string
	stdoutNamedPipe string
	stderrNamedPipe string
)

var (
	serverWg sync.WaitGroup
	clientWg sync.WaitGroup
)

func AddCmdlineToCobra(rootCmd *cobra.Command) {
	rootCmd.Flags().BoolVarP(
		&IsElevateMode, "elevate", "", false,
		"elevate mode (internal)",
	)
	_ = rootCmd.Flags().MarkHidden("elevate")

	rootCmd.Flags().StringVarP(
		&stdinNamedPipe, "stdin", "", "",
		"stdin named pipe (internal)",
	)
	_ = rootCmd.Flags().MarkHidden("stdin")

	rootCmd.Flags().StringVarP(
		&stdoutNamedPipe, "stdout", "", "",
		"stdout named pipe (internal)",
	)
	_ = rootCmd.Flags().MarkHidden("stdout")

	rootCmd.Flags().StringVarP(
		&stderrNamedPipe, "stderr", "", "",
		"stderr named pipe (internal)",
	)
	_ = rootCmd.Flags().MarkHidden("stderr")
}

func GenPipeName() string {
	return uuid.New().String()
}

func handleClient(c net.Conn, f *os.File, directOut bool) {
	defer c.Close()
	if directOut {
		reader := bufio.NewReader(c)
		reader.WriteTo(f)
	} else {
		reader := bufio.NewReader(f)
		reader.WriteTo(c)
	}
}

func serverConnectIO(pipePath string, f *os.File, directOut bool) {
	pipePath = fmt.Sprintf(`\\.\pipe\%s`, pipePath)
	l, err := winio.ListenPipe(pipePath, nil)
	if err != nil {
		log.Fatal("listen error:", err)
	}

	serverWg.Done()
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("accept error:", err)
		}
		go handleClient(conn, f, directOut)
	}
}

func clientConnectIO(pipePath string, f **os.File, directOut bool) {
	pipePath = fmt.Sprintf(`\\.\pipe\%s`, pipePath)
	c, err := winio.DialPipe(pipePath, nil)
	if err != nil {
		log.Fatalf("error opening pipe: %v", err)
	}

	if directOut {
		r, w, _ := os.Pipe()
		*f = w
		reader := bufio.NewReader(r)
		clientWg.Done()
		reader.WriteTo(c)
	} else {
		r, w, _ := os.Pipe()
		*f = r
		reader := bufio.NewReader(c)
		clientWg.Done()
		reader.WriteTo(w)
	}
}

func ConnectClient() {
	clientWg = sync.WaitGroup{}
	clientWg.Add(3)
	go clientConnectIO(stdinNamedPipe, &os.Stdin, false)
	go clientConnectIO(stdoutNamedPipe, &os.Stdout, true)
	go clientConnectIO(stderrNamedPipe, &os.Stderr, true)
	clientWg.Wait()

	// update log default writer
	log.SetOutput(os.Stderr)
}

func RunAsElevated() {
	stdinPipeName := GenPipeName()
	stdoutPipeName := GenPipeName()
	stderrPipeName := GenPipeName()

	verb := "runas"
	exe, _ := os.Executable()
	cwd, _ := os.Getwd()

	argList := []string{"--elevate", "--stdin", stdinPipeName, "--stdout", stdoutPipeName, "--stderr", stderrPipeName}
	argList = append(argList, os.Args[1:]...)
	args := strings.Join(argList, " ")

	serverWg = sync.WaitGroup{}
	serverWg.Add(3)
	go serverConnectIO(stdinPipeName, os.Stdin, false)
	go serverConnectIO(stdoutPipeName, os.Stdout, true)
	go serverConnectIO(stderrPipeName, os.Stderr, true)
	serverWg.Wait()

	err := _ShellExecuteAndWait(0, verb, exe, args, cwd, SW_HIDE)

	if err != nil {
		log.Fatal("shell execute error:", err)
	}
}

func Run(cmd *cobra.Command, args []string, fn func(*cobra.Command, []string)) {
	if !IsAdmin() {
		RunAsElevated()
	} else if IsElevateMode {
		ConnectClient() // connect to pipe
		fn(cmd, args)
		time.Sleep(100 * time.Millisecond)
	} else {
		fn(cmd, args) // we are already admin
	}
}

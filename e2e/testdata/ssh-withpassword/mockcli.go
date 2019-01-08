package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const (
	ptyMaster = "/dev/ptmx"
)

func main() {
	dockerUser := os.Getenv("MOCK_DOCKER_USER")
	if len(dockerUser) == 0 {
		dockerUser = "penguin"
	}
	dockerHostSSHPassword := os.Getenv("MOCK_DOCKER_HOST_SSH_PASSWORD")
	if len(dockerHostSSHPassword) == 0 {
		dockerHostSSHPassword = "mmNNTTbb140821"
	}
	cmd := exec.Command("docker-zhm", os.Args[1:]...)
	cmd.Env = os.Environ()
	p, err := start(cmd)
	if err != nil {
		fmt.Printf("docker-zhm start error:%v", err)
		os.Exit(1)
	}

	go func(p *os.File) {
		sshp := 0
		for {
			buf := make([]byte, 1024)
			n, err := p.Read(buf)
			if err != nil {
				break
			}
			str := string(buf[:n])
			if len(str) > 0 {
				if strings.HasPrefix(str, dockerUser) && strings.HasSuffix(str, "'s password: ") {
					p.Write([]byte(fmt.Sprintf("%s\r", dockerHostSSHPassword)))
					sshp = 1
				} else {
					if sshp == 0 {
						fmt.Print(strings.Replace(str, "\r\n", "\n", -1))
					} else {
						sshp = 0
					}
				}
			}
		}
	}(p)

	go func() {
		for {
			buf := make([]byte, 1024)
			rd := bufio.NewReader(os.Stdin)
			n, err := rd.Read(buf)
			if err != nil {
				break
			}
			p.Write(buf[:n])
		}
	}()

	if err := cmd.Wait(); err != nil {
		str := cmd.ProcessState.String()
		exitCode := strings.Replace(str, "exit status ", "", -1)
		exitCode = strings.Replace(exitCode, "exit status: ", "", -1)
		e, _ := strconv.Atoi(exitCode)
		os.Exit(e)
	}
}

func start(cmd *exec.Cmd) (pty *os.File, err error) {
	m, s, e := getPty()
	if e != nil {
		return nil, e
	}
	defer s.Close()
	cmd.Stdin = s
	cmd.Stdout = s
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setctty: true, Setsid: true}
	err = cmd.Start()
	if err != nil {
		m.Close()
		return nil, err
	}
	return m, err
}

func getPty() (master, slave *os.File, err error) {
	m, e := os.OpenFile(ptyMaster, os.O_RDWR, 0)
	if e != nil {
		return nil, nil, e
	}

	sname, e := ptsName(m)
	if e != nil {
		return nil, nil, e
	}

	e = unlockpt(m)

	if e != nil {
		return nil, nil, e
	}

	s, e := os.OpenFile(sname, os.O_RDWR, 0)
	if e != nil {
		return nil, nil, e
	}
	return m, s, nil
}

func ioctl(fd, cmd, ptr uintptr) error {
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, cmd, ptr)
	if e != 0 {
		return e
	}
	return nil
}

func ptsName(f *os.File) (string, error) {
	var n int
	if err := ioctl(f.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n))); err != nil {
		return "", err
	}
	return fmt.Sprintf("/dev/pts/%d", n), nil
}

func unlockpt(f *os.File) error {
	var n int
	return ioctl(f.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&n)))
}

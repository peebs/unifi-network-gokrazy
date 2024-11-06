package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/gokrazy/gokrazy"
)

func podman(args ...string) error {
	podman := exec.Command("/usr/local/bin/podman", args...)
	podman.Env = expandPath(os.Environ())
	podman.Env = append(podman.Env, "TMPDIR=/tmp")
	podman.Stdin = os.Stdin
	podman.Stdout = os.Stdout
	podman.Stderr = os.Stderr
	if err := podman.Run(); err != nil {
		return fmt.Errorf("%v: %v", podman.Args, err)
	}
	return nil
}

func unifiController() error {
	// Ensure we have an up-to-date clock, which in turn also means that
	// networking is up. This is relevant because podman takes what’s in
	// /etc/resolv.conf (nothing at boot) and holds on to it, meaning your
	// container will never have working networking if it starts too early.
	gokrazy.WaitForClock()

	if err := podman("kill", "unifi"); err != nil {
		log.Print(err)
	}

	if err := podman("rm", "unifi"); err != nil {
		log.Print(err)
	}

	// You could podman pull here.

	if err := podman("run",
		"--rm",
		"-p", "8080:8080",
		"-p", "8443:8443",
		"-p", "3478:3478/udp",
		"-p", "10001:10001/udp",
		"-v", "/perm/home/unifi-controller-gokrazy:/unifi",
		"-e", "TZ=America/Los_Angeles",
		"-e", "LOTSOFDEVICES=true",
		"-e", "RUNAS_UID0=false",
		"-e", "UNIFI_UID=1000",
		"-e", "UNIFI_GID=1000",
		// "--network", "host",
		"--name", "unifi",
		"jacobalberty/unifi:latest"); err != nil {
		return err
	}

	return nil
}

func main() {
	if err := unifiController(); err != nil {
		log.Fatal(err)
	}
}

// expandPath returns env, but with PATH= modified or added
// such that both /user and /usr/local/bin are included, which podman needs.
func expandPath(env []string) []string {
	extra := "/user:/usr/local/bin"
	found := false
	for idx, val := range env {
		parts := strings.Split(val, "=")
		if len(parts) < 2 {
			continue // malformed entry
		}
		key := parts[0]
		if key != "PATH" {
			continue
		}
		val := strings.Join(parts[1:], "=")
		env[idx] = fmt.Sprintf("%s=%s:%s", key, extra, val)
		found = true
	}
	if !found {
		const busyboxDefaultPATH = "/usr/local/sbin:/sbin:/usr/sbin:/usr/local/bin:/bin:/usr/bin"
		env = append(env, fmt.Sprintf("PATH=%s:%s", extra, busyboxDefaultPATH))
	}
	return env
}

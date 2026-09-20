//go:build ignore

package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const releaseURL = "https://api.github.com/repos/pytgcalls/ntgcalls/releases/tags/v2.2.5"

type Release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func main() {
	start := time.Now()
	if err := setup(); err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("\nSetup completed successfully!")
	fmt.Printf("Time elapsed: %v\n", time.Since(start))
}

func setup() error {
	fmt.Printf("Looking for %s/%s static build...\n", runtime.GOOS, runtime.GOARCH)
	var release Release
	resp, err := http.Get(releaseURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub returned %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return err
	}
	fmt.Println("Latest release:", release.TagName)
	goos, goarch := runtime.GOOS, runtime.GOARCH
	if goos == "darwin" {
		goos = "macos"
	}
	if goarch == "amd64" {
		goarch = "x86_64"
	}
	expected := fmt.Sprintf("ntgcalls.%s-%s-static_libs.zip", goos, goarch)
	var downloadURL string
	for _, a := range release.Assets {
		if strings.EqualFold(a.Name, expected) {
			downloadURL = a.URL
			fmt.Println("Found:", a.Name)
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("no static build for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	fmt.Println("Downloading:", expected)
	resp, err = http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	f, err := os.Create("ntgcalls.zip")
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	f.Close()
	defer os.Remove("ntgcalls.zip")
	fmt.Println("Extracting...")
	r, err := zip.OpenReader("ntgcalls.zip")
	if err != nil {
		return err
	}
	defer r.Close()
	tempDir := "ntgcalls_tmp"
	_ = os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)
	for _, file := range r.File {
		target := filepath.Join(tempDir, file.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(tempDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid path %s", file.Name)
		}
		if file.FileInfo().IsDir() {
			_ = os.MkdirAll(target, file.Mode())
			continue
		}
		_ = os.MkdirAll(filepath.Dir(target), 0755)
		in, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			in.Close()
			return err
		}
		_, err = io.Copy(out, in)
		in.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	copied := 0
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return walkErr
		}
		name := filepath.Base(path)
		var dest string
		switch {
		case name == "ntgcalls.h":
			dest = filepath.Join("ntgcalls", name)
		case strings.HasPrefix(name, "libntgcalls.") || strings.HasPrefix(name, "ntgcalls."):
			dest = name
		default:
			return nil
		}
		_ = os.MkdirAll(filepath.Dir(dest), 0755)
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(dest)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, in)
		out.Close()
		if err != nil {
			return err
		}
		fmt.Println("  copied", dest)
		copied++
		return nil
	})
	if err != nil {
		return err
	}
	if copied == 0 {
		return fmt.Errorf("no ntgcalls files in archive")
	}
	if runtime.GOOS == "linux" {
		fmt.Println("Injecting resolv stubs into libntgcalls.a...")
		if err := injectResolvStub(); err != nil {
			return err
		}
		fmt.Println("  resolv stub injected")
	}
	return nil
}

func injectResolvStub() error {
	stubC := "#include <stddef.h>\n#include <string.h>\nint __dn_expand(const unsigned char *msg, const unsigned char *eom, const unsigned char *src, char *dst, int dstsiz){ if(!src||!dst||dstsiz<=0) return -1; strncpy(dst,(const char*)src,dstsiz-1); dst[dstsiz-1]=0; return (int)strlen((const char*)src)+1; }\nint __res_nquery(void *statp, const char *dname, int cls, int type, unsigned char *answer, int anslen){ return -1; }\n"
	if err := os.WriteFile("/tmp/ntgcalls_resolv_stub.c", []byte(stubC), 0644); err != nil {
		return err
	}
	defer os.Remove("/tmp/ntgcalls_resolv_stub.c")
	defer os.Remove("/tmp/ntgcalls_resolv_stub.o")
	cmd := exec.Command("gcc", "-c", "/tmp/ntgcalls_resolv_stub.c", "-o", "/tmp/ntgcalls_resolv_stub.o")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = exec.Command("ar", "r", "libntgcalls.a", "/tmp/ntgcalls_resolv_stub.o")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

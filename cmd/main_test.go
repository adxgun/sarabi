package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

func TestMain_(t *testing.T) {
	// 7db66f1f-6cf5-40ad-ad6c-47e1e53fac51.sql
	// 8cd03e592624

	f := "7db66f1f-6cf5-40ad-ad6c-47e1e53fac51.sql"
	c := "8cd03e592624"
	fc := fmt.Sprintf("%s:/tmp/%s", c, f)
	cmd := exec.CommandContext(context.Background(), "docker", "cp", fc, f)
	if err := cmd.Run(); err != nil {
		t.Log(err)
		t.Fatal(err)
	}

	fi, err := os.Stat(f)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(fi.Size())
	t.Log(fi.Name())
}

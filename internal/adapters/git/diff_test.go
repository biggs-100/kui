package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitDiffParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
		wantPath string
		wantAdd  int
		wantDel  int
	}{
		{
			name: "single file modified",
			input: `diff --git a/main.go b/main.go
index 1234567..abcdef0 100644
--- a/main.go
+++ b/main.go
@@ -10,7 +10,8 @@ package main
 
 import (
 	"fmt"
+	"strings"
 )
 
 func main() {
-	fmt.Println("hello")
+	fmt.Println(strings.ToUpper("hello"))
 }
`,
			wantLen:  1,
			wantPath: "main.go",
			wantAdd:  2,
			wantDel:  1,
		},
		{
			name: "multiple files",
			input: `diff --git a/foo.go b/foo.go
index 1111111..2222222 100644
--- a/foo.go
+++ b/foo.go
@@ -1,3 +1,4 @@
 package foo
 
+var x = 1
 func Foo() {}
diff --git a/bar.go b/bar.go
index 3333333..4444444 100644
--- a/bar.go
+++ b/bar.go
@@ -5,6 +5,7 @@
 package bar
 
+import "fmt"
 func Bar() {}
`,
			wantLen:  2,
			wantPath: "foo.go",
			wantAdd:  1,
			wantDel:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := parseDiff(tt.input)
			if err != nil {
				t.Fatalf("parseDiff() error = %v", err)
			}
			if len(files) != tt.wantLen {
				t.Fatalf("parseDiff() returned %d files, want %d", len(files), tt.wantLen)
			}
			if files[0].Path != tt.wantPath {
				t.Errorf("files[0].Path = %q, want %q", files[0].Path, tt.wantPath)
			}
			if files[0].Additions != tt.wantAdd {
				t.Errorf("files[0].Additions = %d, want %d", files[0].Additions, tt.wantAdd)
			}
			if files[0].Deletions != tt.wantDel {
				t.Errorf("files[0].Deletions = %d, want %d", files[0].Deletions, tt.wantDel)
			}
		})
	}
}

func TestGitDiffParseHunks(t *testing.T) {
	input := `diff --git a/main.go b/main.go
index 1234567..abcdef0 100644
--- a/main.go
+++ b/main.go
@@ -10,7 +10,8 @@ package main
 
 import (
 	"fmt"
+	"strings"
 )
 
 func main() {
-	fmt.Println("hello")
+	fmt.Println(strings.ToUpper("hello"))
 }
`
	files, err := parseDiff(input)
	if err != nil {
		t.Fatalf("parseDiff() error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	file := files[0]
	if len(file.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(file.Hunks))
	}

	hunk := file.Hunks[0]
	if hunk.OldStart != 10 {
		t.Errorf("hunk.OldStart = %d, want 10", hunk.OldStart)
	}
	if hunk.NewStart != 10 {
		t.Errorf("hunk.NewStart = %d, want 10", hunk.NewStart)
	}
	// Context + additions + deletions
	if len(hunk.Lines) == 0 {
		t.Fatal("expected non-empty hunk lines")
	}

	// Count line types
	var adds, dels, ctx int
	for _, line := range hunk.Lines {
		switch line.Type {
		case "added":
			adds++
		case "removed":
			dels++
		case "context":
			ctx++
		}
	}
	if adds != 2 {
		t.Errorf("added lines = %d, want 2", adds)
	}
	if dels != 1 {
		t.Errorf("removed lines = %d, want 1", dels)
	}
	if ctx == 0 {
		t.Error("expected at least one context line")
	}
}

func TestGitDiffParseNewFile(t *testing.T) {
	input := `diff --git a/new.go b/new.go
new file mode 100644
index 0000000..abcdef0
--- /dev/null
+++ b/new.go
@@ -0,0 +1,3 @@
+package new
+
+func New() {}
`
	files, err := parseDiff(input)
	if err != nil {
		t.Fatalf("parseDiff() error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Status != "added" {
		t.Errorf("status = %q, want %q", files[0].Status, "added")
	}
	if files[0].Additions != 3 {
		t.Errorf("additions = %d, want 3", files[0].Additions)
	}
}

func TestGitDiffParseDeletedFile(t *testing.T) {
	input := `diff --git a/old.go b/old.go
deleted file mode 100644
index abcdef0..0000000
--- a/old.go
+++ /dev/null
@@ -1,3 +0,0 @@
-package old
-
-func Old() {}
`
	files, err := parseDiff(input)
	if err != nil {
		t.Fatalf("parseDiff() error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Status != "deleted" {
		t.Errorf("status = %q, want %q", files[0].Status, "deleted")
	}
	if files[0].Deletions != 3 {
		t.Errorf("deletions = %d, want 3", files[0].Deletions)
	}
}

func TestGitDiffParseEmpty(t *testing.T) {
	files, err := parseDiff("")
	if err != nil {
		t.Fatalf("parseDiff() error = %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files for empty diff, got %d", len(files))
	}
}

func TestGitDiffCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a temp git repo
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Create a file and commit
	writeFile(t, filepath.Join(tmpDir, "hello.txt"), "hello\n")
	gitExec(t, tmpDir, "add", "hello.txt")
	gitExec(t, tmpDir, "commit", "-m", "initial commit", "--allow-empty")

	// Modify the file
	writeFile(t, filepath.Join(tmpDir, "hello.txt"), "hello world\n")

	// Run DiffCommand
	files, err := DiffCommand(tmpDir)
	if err != nil {
		t.Fatalf("DiffCommand() error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("DiffCommand() returned %d files, want 1", len(files))
	}
	if files[0].Path != "hello.txt" {
		t.Errorf("file path = %q, want %q", files[0].Path, "hello.txt")
	}
}

// --- helpers ---

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	gitExec(t, dir, "init")
	gitExec(t, dir, "config", "user.email", "test@test.com")
	gitExec(t, dir, "config", "user.name", "Test")
}

func gitExec(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

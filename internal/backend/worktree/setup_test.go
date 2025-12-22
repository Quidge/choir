package worktree

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quidge/choir/internal/backend"
	"github.com/Quidge/choir/internal/config"
)

func TestHostSetupRunner_Run(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "setup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	runner := &HostSetupRunner{WorkDir: tmpDir}
	ctx := context.Background()

	cfg := &backend.SetupConfig{
		Environment: map[string]string{
			"FOO": "bar",
			"BAZ": "qux",
		},
	}

	if err := runner.Run(ctx, cfg); err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Verify env file was created
	envPath := filepath.Join(tmpDir, envFile)
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Error("env file was not created")
	}

	// Verify contents
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read env file: %v", err)
	}

	if !strings.Contains(string(content), "export FOO='bar'") {
		t.Error("env file missing FOO variable")
	}
	if !strings.Contains(string(content), "export BAZ='qux'") {
		t.Error("env file missing BAZ variable")
	}
}

func TestHostSetupRunner_WriteEnvironment(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "env-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	runner := &HostSetupRunner{WorkDir: tmpDir}

	env := map[string]string{
		"SIMPLE":      "value",
		"WITH_QUOTES": "it's got quotes",
		"EMPTY":       "",
	}

	if err := runner.writeEnvironment(env); err != nil {
		t.Fatalf("writeEnvironment() failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, envFile))
	if err != nil {
		t.Fatalf("failed to read env file: %v", err)
	}

	// Check single quote escaping
	if !strings.Contains(string(content), `export WITH_QUOTES='it'\''s got quotes'`) {
		t.Errorf("single quote not properly escaped in: %s", content)
	}
}

func TestHostSetupRunner_WriteEnvironmentEmpty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "env-empty-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	runner := &HostSetupRunner{WorkDir: tmpDir}

	// Empty environment should not create file
	if err := runner.writeEnvironment(nil); err != nil {
		t.Fatalf("writeEnvironment(nil) failed: %v", err)
	}

	envPath := filepath.Join(tmpDir, envFile)
	if _, err := os.Stat(envPath); !os.IsNotExist(err) {
		t.Error("env file should not be created for empty environment")
	}
}

func TestHostSetupRunner_HandleFilesSymlink(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "files-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source file
	srcDir := filepath.Join(tmpDir, "src")
	os.Mkdir(srcDir, 0755)
	srcFile := filepath.Join(srcDir, "test.txt")
	os.WriteFile(srcFile, []byte("test content"), 0644)

	workDir := filepath.Join(tmpDir, "work")
	os.Mkdir(workDir, 0755)

	runner := &HostSetupRunner{WorkDir: workDir}

	files := []config.FileMount{
		{
			Source:   srcFile,
			Target:   "linked.txt",
			ReadOnly: true,
		},
	}

	if err := runner.handleFiles(files); err != nil {
		t.Fatalf("handleFiles() failed: %v", err)
	}

	// Verify symlink was created
	targetPath := filepath.Join(workDir, "linked.txt")
	info, err := os.Lstat(targetPath)
	if err != nil {
		t.Fatalf("failed to stat target: %v", err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink, got regular file")
	}

	// Verify symlink points to correct source
	linkDest, err := os.Readlink(targetPath)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}
	if linkDest != srcFile {
		t.Errorf("symlink points to %q, expected %q", linkDest, srcFile)
	}
}

func TestHostSetupRunner_HandleFilesCopy(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "files-copy-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source file
	srcDir := filepath.Join(tmpDir, "src")
	os.Mkdir(srcDir, 0755)
	srcFile := filepath.Join(srcDir, "test.txt")
	os.WriteFile(srcFile, []byte("test content"), 0644)

	workDir := filepath.Join(tmpDir, "work")
	os.Mkdir(workDir, 0755)

	runner := &HostSetupRunner{WorkDir: workDir}

	files := []config.FileMount{
		{
			Source:   srcFile,
			Target:   "copied.txt",
			ReadOnly: false, // Should copy instead of symlink
		},
	}

	if err := runner.handleFiles(files); err != nil {
		t.Fatalf("handleFiles() failed: %v", err)
	}

	// Verify file was copied (not symlinked)
	targetPath := filepath.Join(workDir, "copied.txt")
	info, err := os.Lstat(targetPath)
	if err != nil {
		t.Fatalf("failed to stat target: %v", err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("expected regular file, got symlink")
	}

	// Verify content
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("expected 'test content', got %q", content)
	}
}

func TestHostSetupRunner_HandleFilesDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "files-dir-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source directory with files
	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("content2"), 0644)

	workDir := filepath.Join(tmpDir, "work")
	os.Mkdir(workDir, 0755)

	runner := &HostSetupRunner{WorkDir: workDir}

	files := []config.FileMount{
		{
			Source:   srcDir,
			Target:   "copied-dir",
			ReadOnly: false,
		},
	}

	if err := runner.handleFiles(files); err != nil {
		t.Fatalf("handleFiles() failed: %v", err)
	}

	// Verify directory structure was copied
	targetDir := filepath.Join(workDir, "copied-dir")
	if _, err := os.Stat(filepath.Join(targetDir, "file1.txt")); os.IsNotExist(err) {
		t.Error("file1.txt was not copied")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "subdir", "file2.txt")); os.IsNotExist(err) {
		t.Error("subdir/file2.txt was not copied")
	}
}

func TestHostSetupRunner_RunCommands(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cmd-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	runner := &HostSetupRunner{WorkDir: tmpDir}
	ctx := context.Background()

	cfg := &backend.SetupConfig{
		SetupCommands: []string{
			"touch created-by-command.txt",
			"echo 'hello' > output.txt",
		},
	}

	if err := runner.Run(ctx, cfg); err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Verify first command executed
	if _, err := os.Stat(filepath.Join(tmpDir, "created-by-command.txt")); os.IsNotExist(err) {
		t.Error("first command did not execute")
	}

	// Verify second command executed
	content, err := os.ReadFile(filepath.Join(tmpDir, "output.txt"))
	if err != nil {
		t.Fatalf("failed to read output.txt: %v", err)
	}
	if !strings.Contains(string(content), "hello") {
		t.Errorf("expected 'hello' in output.txt, got %q", content)
	}
}

func TestHostSetupRunner_RunCommandsWithEnv(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cmd-env-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	runner := &HostSetupRunner{WorkDir: tmpDir}
	ctx := context.Background()

	cfg := &backend.SetupConfig{
		Environment: map[string]string{
			"MY_VAR": "my_value",
		},
		SetupCommands: []string{
			"echo $MY_VAR > env-output.txt",
		},
	}

	if err := runner.Run(ctx, cfg); err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "env-output.txt"))
	if err != nil {
		t.Fatalf("failed to read env-output.txt: %v", err)
	}
	if !strings.Contains(string(content), "my_value") {
		t.Errorf("expected 'my_value' in output, got %q", content)
	}
}

func TestHostSetupRunner_RunCommandFails(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cmd-fail-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	runner := &HostSetupRunner{WorkDir: tmpDir}
	ctx := context.Background()

	cfg := &backend.SetupConfig{
		SetupCommands: []string{
			"exit 1",
		},
	}

	err = runner.Run(ctx, cfg)
	if err == nil {
		t.Fatal("expected error for failing command")
	}
	if !strings.Contains(err.Error(), "command 1 failed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestHostSetupRunner_ContextCancellation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ctx-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	runner := &HostSetupRunner{WorkDir: tmpDir}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	cfg := &backend.SetupConfig{
		Environment: map[string]string{
			"TEST": "value",
		},
	}

	err = runner.Run(ctx, cfg)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestHostSetupRunner_MissingWorkDir(t *testing.T) {
	runner := &HostSetupRunner{}
	ctx := context.Background()

	cfg := &backend.SetupConfig{}

	err := runner.Run(ctx, cfg)
	if err == nil {
		t.Fatal("expected error for missing work directory")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "copyfile-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")

	// Create source with specific permissions
	if err := os.WriteFile(src, []byte("test content"), 0755); err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile() failed: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read dst: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("expected 'test content', got %q", content)
	}

	// Verify permissions were preserved
	srcInfo, _ := os.Stat(src)
	dstInfo, _ := os.Stat(dst)
	if srcInfo.Mode() != dstInfo.Mode() {
		t.Errorf("permissions not preserved: src=%v, dst=%v", srcInfo.Mode(), dstInfo.Mode())
	}
}

func TestCopyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "copydir-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	src := filepath.Join(tmpDir, "src")
	dst := filepath.Join(tmpDir, "dst")

	// Create directory structure
	os.MkdirAll(filepath.Join(src, "a", "b"), 0755)
	os.WriteFile(filepath.Join(src, "root.txt"), []byte("root"), 0644)
	os.WriteFile(filepath.Join(src, "a", "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(src, "a", "b", "b.txt"), []byte("b"), 0644)

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir() failed: %v", err)
	}

	// Verify structure
	files := []string{
		filepath.Join(dst, "root.txt"),
		filepath.Join(dst, "a", "a.txt"),
		filepath.Join(dst, "a", "b", "b.txt"),
	}

	for _, f := range files {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("file not copied: %s", f)
		}
	}

	// Verify content
	content, _ := os.ReadFile(filepath.Join(dst, "a", "b", "b.txt"))
	if string(content) != "b" {
		t.Errorf("expected 'b', got %q", content)
	}
}

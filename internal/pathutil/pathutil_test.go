package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "empty path",
			path: "",
			want: "",
		},
		{
			name: "just tilde",
			path: "~",
			want: home,
		},
		{
			name: "tilde with subpath",
			path: "~/Documents/test",
			want: filepath.Join(home, "Documents/test"),
		},
		{
			name: "absolute path unchanged",
			path: "/usr/local/bin",
			want: "/usr/local/bin",
		},
		{
			name: "relative path unchanged",
			path: "relative/path",
			want: "relative/path",
		},
		{
			name: "tilde in middle unchanged",
			path: "/home/~user/test",
			want: "/home/~user/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandTilde(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandTilde() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExpandTilde() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveRelative(t *testing.T) {
	tests := []struct {
		name string
		base string
		path string
		want string
	}{
		{
			name: "relative path",
			base: "/home/user/project",
			path: "src/main.go",
			want: "/home/user/project/src/main.go",
		},
		{
			name: "absolute path unchanged",
			base: "/home/user/project",
			path: "/etc/config",
			want: "/etc/config",
		},
		{
			name: "relative with parent refs",
			base: "/home/user/project",
			path: "../other/file",
			want: "/home/user/other/file",
		},
		{
			name: "dot path",
			base: "/home/user/project",
			path: "./config",
			want: "/home/user/project/config",
		},
		{
			name: "empty path",
			base: "/home/user/project",
			path: "",
			want: "/home/user/project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveRelative(tt.base, tt.path)
			if got != tt.want {
				t.Errorf("ResolveRelative() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAbsolute(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "absolute path",
			path: "/home/user",
			want: true,
		},
		{
			name: "relative path",
			path: "relative/path",
			want: false,
		},
		{
			name: "empty path",
			path: "",
			want: false,
		},
		{
			name: "dot path",
			path: "./config",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAbsolute(tt.path)
			if got != tt.want {
				t.Errorf("IsAbsolute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateAbsolute(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "absolute path",
			path:    "/home/user",
			wantErr: false,
		},
		{
			name:    "relative path",
			path:    "relative/path",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAbsolute(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAbsolute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExists(t *testing.T) {
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "testfile")
	if err := os.WriteFile(tmpPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("existing file", func(t *testing.T) {
		if !Exists(tmpPath) {
			t.Error("Exists() = false, want true for existing file")
		}
	})

	t.Run("existing directory", func(t *testing.T) {
		if !Exists(tmpDir) {
			t.Error("Exists() = false, want true for existing directory")
		}
	})

	t.Run("non-existing path", func(t *testing.T) {
		if Exists("/nonexistent/path/to/file") {
			t.Error("Exists() = true, want false for non-existing path")
		}
	})
}

func TestExistsAndIsDir(t *testing.T) {
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "testfile")
	if err := os.WriteFile(tmpPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("directory", func(t *testing.T) {
		if !ExistsAndIsDir(tmpDir) {
			t.Error("ExistsAndIsDir() = false, want true for directory")
		}
	})

	t.Run("file", func(t *testing.T) {
		if ExistsAndIsDir(tmpPath) {
			t.Error("ExistsAndIsDir() = true, want false for file")
		}
	})

	t.Run("non-existing", func(t *testing.T) {
		if ExistsAndIsDir("/nonexistent/path") {
			t.Error("ExistsAndIsDir() = true, want false for non-existing path")
		}
	})
}

func TestExistsAndIsFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "testfile")
	if err := os.WriteFile(tmpPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("file", func(t *testing.T) {
		if !ExistsAndIsFile(tmpPath) {
			t.Error("ExistsAndIsFile() = false, want true for file")
		}
	})

	t.Run("directory", func(t *testing.T) {
		if ExistsAndIsFile(tmpDir) {
			t.Error("ExistsAndIsFile() = true, want false for directory")
		}
	})

	t.Run("non-existing", func(t *testing.T) {
		if ExistsAndIsFile("/nonexistent/path") {
			t.Error("ExistsAndIsFile() = true, want false for non-existing path")
		}
	})
}

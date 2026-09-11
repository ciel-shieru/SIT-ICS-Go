package environment

import (
	"os"
	"sync"
	"testing"
	"time"
)

type fakeFS struct {
	mu        sync.RWMutex
	files     map[string][]byte
	dirs      map[string]bool
	envVars   map[string]string
	statErrs  map[string]error
	readErrs  map[string]error
}

func (f *fakeFS) Stat(name string) (os.FileInfo, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if err, ok := f.statErrs[name]; ok {
		return nil, err
	}
	if f.dirs[name] {
		return &fakeFileInfo{name: name}, nil
	}
	return nil, os.ErrNotExist
}

type fakeFileInfo struct {
	name string
}

func (f *fakeFileInfo) Name() string       { return f.name }
func (f *fakeFileInfo) Size() int64        { return 0 }
func (f *fakeFileInfo) Mode() os.FileMode  { return 0 }
func (f *fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f *fakeFileInfo) IsDir() bool        { return false }
func (f *fakeFileInfo) Sys() interface{}   { return nil }
func (f *fakeFileInfo) Info() (os.FileInfo, error) { return f, nil }

func (f *fakeFS) ReadFile(name string) ([]byte, error) {
	if err, ok := f.readErrs[name]; ok {
		return nil, err
	}
	if content, ok := f.files[name]; ok {
		return content, nil
	}
	return nil, os.ErrNotExist
}

func (f *fakeFS) LookupEnv(key string) (string, bool) {
	val, ok := f.envVars[key]
	return val, ok
}

func newFakeFS(opts ...func(*fakeFS)) *fakeFS {
	f := &fakeFS{
		files:   make(map[string][]byte),
		dirs:    make(map[string]bool),
		envVars: make(map[string]string),
		statErrs: make(map[string]error),
		readErrs: make(map[string]error),
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func withFile(path string, content []byte) func(*fakeFS) {
	return func(f *fakeFS) {
		f.files[path] = content
	}
}

func withDir(path string) func(*fakeFS) {
	return func(f *fakeFS) {
		f.dirs[path] = true
	}
}

func withEnv(key, value string) func(*fakeFS) {
	return func(f *fakeFS) {
		f.envVars[key] = value
	}
}

func withStatErr(path string, err error) func(*fakeFS) {
	return func(f *fakeFS) {
		f.statErrs[path] = err
	}
}

func withReadErr(path string, err error) func(*fakeFS) {
	return func(f *fakeFS) {
		f.readErrs[path] = err
	}
}

func TestDetectContainerized(t *testing.T) {
	tests := []struct {
		name  string
		setup func() *fakeFS
		want  bool
	}{
		{
			name:  "no signals",
			setup: func() *fakeFS { return newFakeFS() },
			want:  false,
		},
		{
			name: "KUBERNETES_SERVICE_HOST set",
			setup: func() *fakeFS {
				return newFakeFS(withEnv("KUBERNETES_SERVICE_HOST", "10.96.0.1"))
			},
			want: true,
		},
		{
			name: "KUBERNETES_SERVICE_HOST empty",
			setup: func() *fakeFS {
				return newFakeFS(withEnv("KUBERNETES_SERVICE_HOST", ""))
			},
			want: false,
		},
		{
			name:  "/.dockerenv present",
			setup: func() *fakeFS { return newFakeFS(withDir("/.dockerenv")) },
			want:  true,
		},
		{
			name:  "/run/.containerenv present",
			setup: func() *fakeFS { return newFakeFS(withDir("/run/.containerenv")) },
			want:  true,
		},
		{
			name: "ServiceAccount namespace present",
			setup: func() *fakeFS {
				return newFakeFS(withDir("/var/run/secrets/kubernetes.io/serviceaccount/namespace"))
			},
			want: true,
		},
		{
			name: "kubepods cgroup v1",
			setup: func() *fakeFS {
				return newFakeFS(withFile("/proc/self/cgroup", []byte("11:cpuset:/kubepods/podabc/cri-dockerdef")))
			},
			want: true,
		},
		{
			name: "docker cgroup v1",
			setup: func() *fakeFS {
				return newFakeFS(withFile("/proc/self/cgroup", []byte("0::/docker/abc123def456\n11:pids:/docker/abc123def456\n")))
			},
			want: true,
		},
		{
			name: "containerd cgroup v1",
			setup: func() *fakeFS {
				return newFakeFS(withFile("/proc/self/cgroup", []byte("0::/containerd/io.containerd.runtime.v2.task/my-container\n")))
			},
			want: true,
		},
		{
			name: "libpod cgroup",
			setup: func() *fakeFS {
				return newFakeFS(withFile("/proc/self/cgroup", []byte("0::/libpod/containerid\n")))
			},
			want: true,
		},
		{
			name: "crio cgroup",
			setup: func() *fakeFS {
				return newFakeFS(withFile("/proc/self/cgroup", []byte("0::/crio/containerid\n")))
			},
			want: true,
		},
		{
			name: "kubepods cgroup v2 slice",
			setup: func() *fakeFS {
				return newFakeFS(withFile("/proc/self/cgroup", []byte("0::/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-podabc.slice\n")))
			},
			want: true,
		},
		{
			name: "kubepods cgroup v2 guaranteed slice",
			setup: func() *fakeFS {
				return newFakeFS(withFile("/proc/self/cgroup", []byte("0::/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod123.slice\n")))
			},
			want: true,
		},
		{
			name: "normal cgroup no marker",
			setup: func() *fakeFS {
				return newFakeFS(withFile("/proc/self/cgroup", []byte("0::/\n11:pids:/system.slice/myapp.service\n")))
			},
			want: false,
		},
		{
			// enableServiceLinks: false still sets KUBERNETES_SERVICE_HOST env var.
			name: "KUBERNETES_SERVICE_HOST set (enableServiceLinks=false)",
			setup: func() *fakeFS {
				return newFakeFS(withEnv("KUBERNETES_SERVICE_HOST", "10.96.0.1"))
			},
			want: true,
		},
		{
			// automountServiceAccountToken: false does not affect KUBERNETES_SERVICE_HOST.
			name: "KUBERNETES_SERVICE_HOST set without ServiceAccount",
			setup: func() *fakeFS {
				return newFakeFS(withEnv("KUBERNETES_SERVICE_HOST", "10.96.0.1"))
			},
			want: true,
		},
		{
			name:  "missing marker files no error",
			setup: func() *fakeFS { return newFakeFS() },
			want:  false,
		},
		{
			name: "unreadable /proc/self/cgroup",
			setup: func() *fakeFS {
				return newFakeFS(withReadErr("/proc/self/cgroup", os.ErrPermission))
			},
			want: false,
		},
		{
			name: "malformed cgroup input",
			setup: func() *fakeFS {
				return newFakeFS(withFile("/proc/self/cgroup", []byte("this is not a valid cgroup file\nrandom garbage \n\n")))
			},
			want: false,
		},
		{
			name: "multiple signals present",
			setup: func() *fakeFS {
				return newFakeFS(
					withEnv("KUBERNETES_SERVICE_HOST", "10.96.0.1"),
					withDir("/.dockerenv"),
				)
			},
			want: true,
		},
		{
			name: "cgroup read error with no other signals",
			setup: func() *fakeFS {
				return newFakeFS(withReadErr("/proc/self/cgroup", os.ErrPermission))
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := tt.setup()
			got := detectWithFS(fs)
			if got != tt.want {
				t.Errorf("detectWithFS() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsContainerized_Caching(t *testing.T) {
	resetForTesting()
	defer resetForTesting()

	originalFS := fs
	defer func() { fs = originalFS }()

	var callCount int
	mu := sync.Mutex{}

	targetFS := &fakeFS{
		files:   make(map[string][]byte),
		dirs:    map[string]bool{"/.dockerenv": true},
		envVars: make(map[string]string),
		statErrs: make(map[string]error),
		readErrs: make(map[string]error),
	}

	countingFs := &countingFS{
		fs: targetFS,
		onStat: func() {
			mu.Lock()
			callCount++
			mu.Unlock()
		},
	}
	fs = countingFs

	_ = IsContainerized()
	_ = IsContainerized()
	_ = IsContainerized()

	mu.Lock()
	count := callCount
	mu.Unlock()

	if count != 1 {
		t.Errorf("detection ran %d times, expected 1 (sync.Once should cache result)", count)
	}
}

type countingFS struct {
	fs     filesystem
	onStat func()
}

func (f *countingFS) Stat(name string) (os.FileInfo, error) {
	if f.onStat != nil {
		f.onStat()
	}
	return f.fs.Stat(name)
}

func (f *countingFS) ReadFile(name string) ([]byte, error) {
	return f.fs.ReadFile(name)
}

func (f *countingFS) LookupEnv(key string) (string, bool) {
	return f.fs.LookupEnv(key)
}

func TestDetectWithFS_PermissionDenied(t *testing.T) {
	fs := newFakeFS(
		withStatErr("/.dockerenv", os.ErrPermission),
		withStatErr("/run/.containerenv", os.ErrPermission),
		withStatErr("/var/run/secrets/kubernetes.io/serviceaccount/namespace", os.ErrPermission),
		withReadErr("/proc/self/cgroup", os.ErrPermission),
	)

	got := detectWithFS(fs)
	if got != false {
		t.Errorf("detectWithFS() = %v, want false (permission denied on all signals)", got)
	}
}

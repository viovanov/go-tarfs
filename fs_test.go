package tarfs

import (
	"io/fs"
	"os"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFS(t *testing.T) {
	require := require.New(t)

	f, err := os.Open("test.tar")
	require.NoError(err)

	tfs, err := New(f)
	require.NoError(err)

	err = fstest.TestFS(tfs, "foo", "dir1/dir11")
	require.NoError(err)
}

func TestOpenInvalid(t *testing.T) {
	require := require.New(t)

	f, err := os.Open("test.tar")
	require.NoError(err)

	tfs, err := New(f)
	require.NoError(err)

	for _, name := range []string{"/foo", "./foo", "foo/", "foo/../foo", "foo//bar"} {
		_, err := tfs.Open(name)
		assert.ErrorIsf(t, err, fs.ErrInvalid, "when tarfs.Open(%#v)", name)
	}
}

func TestOpenNotExist(t *testing.T) {
	require := require.New(t)

	f, err := os.Open("test.tar")
	require.NoError(err)
	defer f.Close()

	tfs, err := New(f)
	require.NoError(err)

	for _, name := range []string{"baz", "qwe", "foo/bar", "file11"} {
		_, err := tfs.Open(name)
		assert.ErrorIsf(t, err, fs.ErrNotExist, "when tarfs.Open(%#v)", name)
	}
}

func TestOpenThenStat(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	f, err := os.Open("test.tar")
	require.NoError(err)
	defer f.Close()

	tfs, err := New(f)
	require.NoError(err)

	for _, file := range []struct {
		path  string
		name  string
		isDir bool
	}{
		{"foo", "foo", false},
		{"bar", "bar", false},
		{"dir1", "dir1", true},
		{"dir1/file11", "file11", false},
		{".", ".", true},
	} {
		f, err := tfs.Open(file.path)
		if !assert.NoErrorf(err, "when tarfs.Open(%#v)", file.path) {
			continue
		}

		fi, err := f.Stat()
		if !assert.NoErrorf(err, "when file{%#v}.Stat()", file.path) {
			continue
		}

		assert.Equalf(file.name, fi.Name(), "file{%#v}.Stat().Name()", file.path)
		assert.Equalf(file.isDir, fi.IsDir(), "file{%#v}.Stat().IsDir()", file.path)
	}
}

func TestReadDir(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	f, err := os.Open("test.tar")
	require.NoError(err)
	defer f.Close()

	tfs, err := New(f)
	require.NoError(err)

	for _, dir := range []struct {
		name       string
		entriesLen int
	}{
		{".", 4},
		{"dir1", 3},
		{"dir2/dir21", 2},
	} {
		entries, err := fs.ReadDir(tfs, dir.name)
		if !assert.NoErrorf(err, "when fs.ReadDir(tfs, %#v)", dir.name) {
			continue
		}

		assert.Equalf(dir.entriesLen, len(entries), "len(entries) for %#v", dir.name)
	}
}

func TestReadDirNotDir(t *testing.T) {
	require := require.New(t)

	f, err := os.Open("test.tar")
	require.NoError(err)
	defer f.Close()

	tfs, err := New(f)
	require.NoError(err)

	for _, name := range []string{"foo", "dir1/file12"} {
		_, err := fs.ReadDir(tfs, name)
		assert.ErrorIsf(t, err, ErrNotDir, "when tarfs.ReadDir(tfs, %#v)", name)
	}
}

func TestReadFile(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	f, err := os.Open("test.tar")
	require.NoError(err)
	defer f.Close()

	tfs, err := New(f)
	require.NoError(err)

	for name, content := range map[string]string{
		"dir1/dir11/file111": "file111",
		"dir2/dir21/file212": "file212",
		"foo":                "foo",
	} {
		b, err := fs.ReadFile(tfs, name)
		if !assert.NoErrorf(err, "when fs.ReadFile(tfs, %#v)", name) {
			continue
		}

		assert.Equalf(content, string(b), "in %#v", name)
	}
}

func TestStat(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	f, err := os.Open("test.tar")
	require.NoError(err)
	defer f.Close()

	tfs, err := New(f)
	require.NoError(err)

	for _, file := range []struct {
		path  string
		name  string
		isDir bool
	}{
		{"dir1/dir11/file111", "file111", false},
		{"foo", "foo", false},
		{"dir2/dir21", "dir21", true},
		{".", ".", true},
	} {
		fi, err := fs.Stat(tfs, file.path)
		if !assert.NoErrorf(err, "when fs.Stat(tfs, %#v)", file.path) {
			continue
		}

		assert.Equalf(file.name, fi.Name(), "FileInfo{%#v}.Name()", file.path)

		assert.Equalf(file.isDir, fi.IsDir(), "FileInfo{%#v}.IsDir()", file.path)
	}
}

func TestGlob(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	f, err := os.Open("test.tar")
	require.NoError(err)
	defer f.Close()

	tfs, err := New(f)
	require.NoError(err)

	for pattern, expected := range map[string][]string{
		"*/*2*":   {"dir1/file12", "dir2/dir21"},
		"*":       {"bar", "dir1", "dir2", "foo"},
		"*/*/*":   {"dir1/dir11/file111", "dir2/dir21/file211", "dir2/dir21/file212"},
		"*/*/*/*": nil,
	} {
		actual, err := fs.Glob(tfs, pattern)
		if !assert.NoErrorf(err, "when fs.Glob(tfs, %#v)", pattern) {
			continue
		}

		assert.ElementsMatchf(expected, actual, "matches for pattern %#v", pattern)
	}
}

func TestSubThenReadDir(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	f, err := os.Open("test.tar")
	require.NoError(err)
	defer f.Close()

	tfs, err := New(f)
	require.NoError(err)

	for _, dir := range []struct {
		name       string
		entriesLen int
	}{
		{".", 4},
		{"dir1", 3},
		{"dir2/dir21", 2},
	} {
		subfs, err := fs.Sub(tfs, dir.name)
		if !assert.NoErrorf(err, "when fs.Sub(tfs, %#v)", dir.name) {
			continue
		}

		entries, err := fs.ReadDir(subfs, ".")
		if !assert.NoErrorf(err, "when fs.ReadDir(subfs, %#v)", dir.name) {
			continue
		}

		assert.Equalf(dir.entriesLen, len(entries), "len(entries) for %#v", dir.name)
	}
}

func TestSubThenReadFile(t *testing.T) {
	require := require.New(t)

	f, err := os.Open("test.tar")
	require.NoError(err)
	defer f.Close()

	tfs, err := New(f)
	require.NoError(err)

	name := "dir2"

	subfs, err := fs.Sub(tfs, name)
	require.NoErrorf(err, "when fs.Sub(tfs, %#v)", name)

	name = "dir21/file211"
	content := "file211"

	b, err := fs.ReadFile(subfs, name)
	require.NoErrorf(err, "when fs.ReadFile(subfs, %#v)", name)

	require.Equalf(content, string(b), "in %#v", name)
}

func TestReadOnDir(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	tf, err := os.Open("test.tar")
	require.NoError(err)
	defer tf.Close()

	tfs, err := New(tf)
	require.NoError(err)

	var dirs = []string{"dir1", "dir2/dir21", "."}

	for _, name := range dirs {
		f, err := tfs.Open(name)
		if !assert.NoErrorf(err, "when fs.ReadFile(subfs, %#v)", name) {
			continue
		}

		_, err = f.Read(make([]byte, 1))
		assert.ErrorIsf(err, ErrDir, "when file{%#v}.Read()", name)

		_, err = fs.ReadFile(tfs, name)
		assert.ErrorIsf(err, ErrDir, "fs.ReadFile(tfs, %#v)", name)
	}
}

func TestWalkDir_WithDotDirInArchive(t *testing.T) {
	require := require.New(t)

	tf, err := os.Open("test-with-dot-dir.tar")
	require.NoError(err)
	defer tf.Close()

	tfs, err := New(tf)
	require.NoError(err)

	paths := make([]string, 0, 12)

	err = fs.WalkDir(tfs, ".", func(path string, d fs.DirEntry, err error) error {
		paths = append(paths, path)
		return nil
	})
	require.NoError(err)

	require.ElementsMatch([]string{
		".",
		"bar",
		"foo",
		"dir1",
		"dir1/dir11",
		"dir1/dir11/file111",
		"dir1/file11",
		"dir1/file12",
		"dir2",
		"dir2/dir21",
		"dir2/dir21/file211",
		"dir2/dir21/file212",
	}, paths)
}

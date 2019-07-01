package bufreader

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/qiniu/log"
	"github.com/qiniu/logkit/conf"
	"github.com/qiniu/logkit/reader"
	. "github.com/qiniu/logkit/reader/config"
	. "github.com/qiniu/logkit/reader/test"
	. "github.com/qiniu/logkit/utils/models"

	"github.com/stretchr/testify/assert"
)

type MockReader struct {
	num int
}

func (m *MockReader) Name() string {
	return "mock"
}
func (m *MockReader) Source() string {
	return "mock"
}

func (m *MockReader) Read(p []byte) (n int, err error) {
	if m.num%1000 == 0 {
		for i, v := range "abchaha\n" {
			p[i] = byte(v)
		}
		m.num++
		return 8, nil
	}
	vv := "abxxxabxxxabxxxabxxxabxxxabxabxxxabxxxabxxxabxxxabxxxabxabxxxabxxxabxxxabxxx\n"
	vv += vv
	for i, v := range vv {
		p[i] = byte(v)
	}
	m.num++
	return len(vv), nil
}
func (m *MockReader) SyncMeta() error {
	return nil
}

func (m *MockReader) Close() error {
	return nil
}

var line string

func BenchmarkReadPattern(b *testing.B) {
	m := &MockReader{}
	c := conf.MapConf{}
	c[KeyLogPath] = "logpath"
	c[KeyMode] = ModeDir
	c[KeyDataSourceTag] = "tag1path"
	ma, err := reader.NewMetaWithConf(c)
	if err != nil {
		b.Error(err)
	}
	r, err := NewReaderSize(m, ma, 1024)
	if err != nil {
		b.Fatal(err)
	}
	err = r.SetMode(ReadModeHeadPatternString, "^abc")
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		line, _ = r.ReadPattern()
	}
}

func Test_BuffReader(t *testing.T) {
	CreateSeqFile(1000, lines)
	defer DestroyDir()
	c := conf.MapConf{
		"log_path":        Dir,
		"meta_path":       MetaDir,
		"mode":            ModeDir,
		"sync_every":      "1",
		"ignore_hidden":   "true",
		"reader_buf_size": "24",
		"read_from":       "oldest",
	}
	r, err := reader.NewFileBufReader(c, false)
	if err != nil {
		t.Error(err)
	}
	rest := []string{}
	for {
		line, err := r.ReadLine()
		if err == nil {
			rest = append(rest, line)
		} else {
			break
		}
	}
	if len(rest) != 12 {
		t.Errorf("rest should be 12, but got %v", len(rest))
	}
	r.Close()
}

func Test_BuffReaderInRunTime(t *testing.T) {
	t.Parallel()
	fileName := filepath.Join(os.TempDir(), "Test_BuffReaderInRunTime")
	//create file & write file
	CreateFile(fileName, "12345\n12345\n12345\n12345\n12345\n12345")
	defer DeleteFile(fileName)
	if time.Now().Second() < 5 {
		time.Sleep(10 * time.Second)
	}
	runTime := strconv.Itoa(time.Now().Hour())
	c := conf.MapConf{
		"log_path":        fileName,
		"meta_path":       MetaDir,
		"mode":            ModeFile,
		"sync_every":      "1",
		"ignore_hidden":   "true",
		"reader_buf_size": "24",
		"read_from":       "oldest",
		"run_time":        runTime + "-",
	}
	r, err := reader.NewFileBufReader(c, false)
	if err != nil {
		t.Error(err)
	}
	rest := []string{}
	for {
		line, err := r.ReadLine()
		if err == nil && line != "" {
			rest = append(rest, line)
		} else {
			break
		}
	}
	if len(rest) != 5 {
		t.Errorf("rest should be 12, but got %v", len(rest))
	}
	r.Close()
}

func Test_BuffReaderOutRunTime(t *testing.T) {
	t.Parallel()
	fileName := filepath.Join(os.TempDir(), "Test_BuffReaderOutRunTime")
	//create file & write file
	CreateFile(fileName, "12345\n12345\n12345\n12345\n12345\n12345")
	defer DeleteFile(fileName)
	runTime := strconv.Itoa(time.Now().Hour() + 2)
	c := conf.MapConf{
		"log_path":        fileName,
		"meta_path":       MetaDir,
		"mode":            ModeFile,
		"sync_every":      "1",
		"ignore_hidden":   "true",
		"reader_buf_size": "24",
		"read_from":       "oldest",
		"run_time":        runTime + "-",
	}
	r, err := reader.NewFileBufReader(c, false)
	if err != nil {
		t.Error(err)
	}
	rest := []string{}
	spaceNum := 0
	for {
		line, err := r.ReadLine()
		if err == nil && line != "" {
			rest = append(rest, line)
		} else {
			if spaceNum > 2 {
				break
			}
			spaceNum++
			continue
		}
	}
	assert.EqualValues(t, 0, len(rest))
	r.Close()
}

func Test_Datasource(t *testing.T) {
	testdir := "Test_Datasource1"
	err := os.Mkdir(testdir, DefaultDirPerm)
	if err != nil {
		log.Error(err)
		return
	}
	defer os.RemoveAll(testdir)

	for _, f := range []string{"f1", "f2", "f3"} {
		file, err := os.OpenFile(filepath.Join(testdir, f), os.O_CREATE|os.O_WRONLY, DefaultFilePerm)
		if err != nil {
			log.Error(err)
			return
		}

		file.WriteString("1234567890\nabc123\n")
		file.Close()
	}
	c := conf.MapConf{
		"log_path":        testdir,
		"mode":            ModeDir,
		"sync_every":      "1",
		"ignore_hidden":   "true",
		"reader_buf_size": "37",
		"read_from":       "oldest",
	}
	r, err := reader.NewFileBufReader(c, false)
	if err != nil {
		t.Error(err)
	}
	var rest []string
	var datasources []string
	for {
		line, err := r.ReadLine()
		if err == nil {
			rest = append(rest, line)
		} else {
			break
		}
		datasources = append(datasources, filepath.Base(r.Source()))
	}
	if len(rest) != 6 {
		t.Errorf("rest should be 6, but got %v", len(rest))
	}
	assert.Equal(t, []string{"f1", "f1", "f2", "f2", "f3", "f3"}, datasources)
	r.Close()
}

func Test_Datasource2(t *testing.T) {
	testdir := "Test_Datasource2"
	err := os.Mkdir(testdir, DefaultDirPerm)
	if err != nil {
		log.Error(err)
		return
	}
	defer os.RemoveAll(testdir)

	for _, f := range []string{"f1", "f2", "f3"} {
		file, err := os.OpenFile(filepath.Join(testdir, f), os.O_CREATE|os.O_WRONLY, DefaultFilePerm)
		if err != nil {
			log.Error(err)
			return
		}

		file.WriteString("1234567890\nabc123\n")
		file.Close()
	}
	c := conf.MapConf{
		"log_path":        testdir,
		"mode":            ModeDir,
		"sync_every":      "1",
		"ignore_hidden":   "true",
		"reader_buf_size": "10",
		"read_from":       "oldest",
	}
	r, err := reader.NewFileBufReader(c, false)
	if err != nil {
		t.Error(err)
	}
	var rest []string
	var datasources []string
	for {
		line, err := r.ReadLine()
		if err == nil {
			rest = append(rest, line)
		} else {
			break
		}
		datasources = append(datasources, filepath.Base(r.Source()))
	}
	if len(rest) != 6 {
		t.Errorf("rest should be 6, but got %v", len(rest))
	}
	assert.Equal(t, []string{"f1", "f1", "f2", "f2", "f3", "f3"}, datasources)
	r.Close()
}

func Test_Datasource3(t *testing.T) {
	testdir := "Test_Datasource3"
	err := os.Mkdir(testdir, DefaultDirPerm)
	if err != nil {
		log.Error(err)
		return
	}
	defer os.RemoveAll(testdir)

	for _, f := range []string{"f1", "f2", "f3"} {
		file, err := os.OpenFile(filepath.Join(testdir, f), os.O_CREATE|os.O_WRONLY, DefaultFilePerm)
		if err != nil {
			log.Error(err)
			return
		}

		file.WriteString("1234567890\nabc123\n")
		file.Close()
	}
	c := conf.MapConf{
		"log_path":        testdir,
		"mode":            ModeDir,
		"sync_every":      "1",
		"ignore_hidden":   "true",
		"reader_buf_size": "20",
		"read_from":       "oldest",
	}
	r, err := reader.NewFileBufReader(c, false)
	if err != nil {
		t.Error(err)
	}
	var rest []string
	var datasources []string
	for {
		line, err := r.ReadLine()
		if err == nil {
			rest = append(rest, line)
		} else {
			break
		}
		datasources = append(datasources, filepath.Base(r.Source()))
	}
	if len(rest) != 6 {
		t.Errorf("rest should be 6, but got %v", len(rest))
	}
	assert.Equal(t, []string{"f1", "f1", "f2", "f2", "f3", "f3"}, datasources)
	r.Close()
}

func Test_BuffReaderBufSizeLarge(t *testing.T) {
	CreateSeqFile(1000, lines)
	defer DestroyDir()
	c := conf.MapConf{
		"log_path":        Dir,
		"meta_path":       MetaDir,
		"mode":            ModeDir,
		"sync_every":      "1",
		"ignore_hidden":   "true",
		"reader_buf_size": "1024",
		"read_from":       "oldest",
	}
	r, err := reader.NewFileBufReader(c, false)
	if err != nil {
		t.Error(err)
	}
	rest := []string{}
	for {
		line, err := r.ReadLine()
		if err == nil {
			rest = append(rest, line)
		} else {
			break
		}
	}
	if len(rest) != 12 {
		t.Errorf("rest should be 12, but got %v", len(rest))
	}
	r.Close()
}

func Test_GBKEncoding(t *testing.T) {
	body := "\x82\x31\x89\x38"
	CreateSeqFile(1000, body)
	exp := "㧯"
	defer DestroyDir()
	c := conf.MapConf{
		"log_path":        Dir,
		"meta_path":       MetaDir,
		"mode":            ModeDir,
		"sync_every":      "1",
		"ignore_hidden":   "true",
		"reader_buf_size": "1024",
		"read_from":       "oldest",
		"encoding":        "gb18030",
	}
	r, err := reader.NewFileBufReader(c, false)
	if err != nil {
		t.Error(err)
	}
	rest := []string{}
	for {
		line, err := r.ReadLine()
		if err == nil {
			rest = append(rest, line)
		} else {
			break
		}
		if line != exp {
			t.Fatalf("should exp %v but got %v", exp, line)
		}
	}
	r.Close()
}

func Test_NoPanicEncoding(t *testing.T) {
	body := "123123"
	CreateSeqFile(1000, body)
	exp := "123123"
	defer DestroyDir()
	c := conf.MapConf{
		"log_path":        Dir,
		"meta_path":       MetaDir,
		"mode":            ModeDir,
		"sync_every":      "1",
		"ignore_hidden":   "true",
		"reader_buf_size": "1024",
		"read_from":       "oldest",
		"encoding":        "nopanic",
	}
	r, err := reader.NewFileBufReader(c, false)
	if err != nil {
		t.Error(err)
	}
	rest := []string{}
	for {
		line, err := r.ReadLine()
		if err == nil {
			rest = append(rest, line)
		} else {
			break
		}
		if line != exp {
			t.Fatalf("should exp %v but got %v", exp, line)
		}
	}
	r.Close()
}

func Test_BuffReaderMultiLine(t *testing.T) {
	body := "test123\n12\n34\n56\ntest\nxtestx\n123\n"
	CreateSeqFile(1000, body)
	defer DestroyDir()
	c := conf.MapConf{
		"log_path":        Dir,
		"meta_path":       MetaDir,
		"mode":            ModeDir,
		"sync_every":      "1",
		"ignore_hidden":   "true",
		"reader_buf_size": "1024",
		"read_from":       "oldest",
		"head_pattern":    "^test*",
	}
	r, err := reader.NewFileBufReader(c, false)
	if err != nil {
		t.Error(err)
	}
	rest := make(map[string]int)
	num := 0
	for {
		line, err := r.ReadLine()
		rest[line]++
		num++

		r.SyncMeta()
		if num > 3 {
			break
		}
		r.SyncMeta()
		assert.NoError(t, err)
	}
	r.Close()
	r, err = reader.NewFileBufReader(c, false)
	if err != nil {
		t.Error(err)
	}
	for {
		line, err := r.ReadLine()
		rest[line]++
		num++

		r.SyncMeta()
		if num >= 6 {
			break
		}
		r.SyncMeta()
		assert.NoError(t, err)
	}
	exp := map[string]int{
		"test123\n12\n34\n56\n": 3,
		"test\nxtestx\n123\n":   3,
	}
	assert.Equal(t, exp, rest)
	r.Close()
}

func Test_BuffReaderStats(t *testing.T) {
	body := "Test_BuffReaderStats\n"
	CreateSeqFile(1000, body)
	defer DestroyDir()
	c := conf.MapConf{
		"log_path":  Dir,
		"meta_path": MetaDir,
		"mode":      ModeDir,
		"read_from": "oldest",
	}
	r, err := reader.NewFileBufReader(c, false)
	if err != nil {
		t.Error(err)
	}
	_, err = r.ReadLine()
	assert.NoError(t, err)
	str, ok := r.(reader.StatsReader)
	assert.Equal(t, true, ok)
	stsx := str.Status()
	expsts := StatsInfo{}
	assert.Equal(t, expsts, stsx)
	r.Close()
}

func Test_FileNotFound(t *testing.T) {
	CreateSeqFile(1000, lines)
	defer DestroyDir()
	c := conf.MapConf{
		"mode":            ModeFile,
		"log_path":        "/home/users/john/log/my.log",
		"meta_path":       MetaDir,
		"sync_every":      "1",
		"ignore_hidden":   "true",
		"reader_buf_size": "24",
		"read_from":       "oldest",
	}
	r, err := reader.NewFileBufReader(c, true)
	assert.Error(t, err)

	c["log_path"] = filepath.Join(Dir, Files[0])
	r, err = reader.NewFileBufReader(c, true)
	assert.NoError(t, err)
	rest := []string{}
	for {
		line, err := r.ReadLine()
		if err == nil {
			rest = append(rest, line)
		} else {
			break
		}
	}
	if len(rest) != 4 {
		t.Errorf("rest should be 4, but got %v", len(rest))
	}
	r.Close()
}

var lines = "123456789\n123456789\n123456789\n123456789\n"

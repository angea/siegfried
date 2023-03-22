package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/richardlehane/siegfried"
	"github.com/richardlehane/siegfried/pkg/config"
	"github.com/richardlehane/siegfried/pkg/pronom"
)

var (
	testhome = flag.String("testhome", "../roy/data", "override the default home directory")
	testdata = flag.String("testdata", filepath.Join(".", "testdata"), "override the default test data directory")
)

var s *siegfried.Siegfried

func setup(opts ...config.Option) error {
	if opts == nil && s != nil {
		return nil
	}
	var err error
	s = siegfried.New()
	config.SetHome(*testhome)
	opts = append(opts, config.SetDoubleUp())
	p, err := pronom.New(opts...)
	if err != nil {
		return err
	}
	return s.Add(p)
}

func identifyT(s *siegfried.Siegfried, p string) ([]string, error) {
	ids := make([]string, 0)
	file, err := os.Open(p)
	if err != nil {
		return nil, fmt.Errorf("failed to open %v, got: %v", p, err)
	}
	t := time.Now()
	c, _ := s.Identify(file, p, "")
	for _, i := range c {
		ids = append(ids, i.String())
	}
	err = file.Close()
	if err != nil {
		return nil, err
	}
	if len(ids) > 10 {
		fmt.Printf("test file %s has %d ids\n", p, len(ids))
	}
	tooLong := time.Millisecond * 500
	elapsed := time.Since(t)
	if elapsed > tooLong {
		fmt.Printf("[WARNING] time to match %s was %s\n", p, elapsed.String())
	}
	return ids, nil
}

func multiIdentifyT(s *siegfried.Siegfried, r string) ([][]string, error) {
	set := make([][]string, 0)
	wf := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if *nr && path != r {
				return filepath.SkipDir
			}
			return nil
		}
		ids, err := identifyT(s, path)
		if err != nil {
			return err
		}
		set = append(set, ids)
		return nil
	}
	err := filepath.Walk(r, wf)
	return set, err
}

func matchString(i []string) string {
	str := "[ "
	for _, v := range i {
		str += v
		str += " "
	}
	return str + "]"
}

func TestSuite(t *testing.T) {
	err := setup()
	if err != nil {
		t.Error(err)
	}
	expect := make([]string, 0)
	names := make([]string, 0)
	wf := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		last := strings.Split(path, string(os.PathSeparator))
		path = last[len(last)-1]
		var idx int
		idx = strings.Index(path, "container")
		if idx < 0 {
			idx = strings.Index(path, "signature")
		}
		if idx < 0 {
			idx = len(path)
		}
		strs := strings.Split(path[:idx-1], "-")
		if len(strs) == 2 {
			expect = append(expect, strings.Join(strs, "/"))
		} else if len(strs) == 3 {
			expect = append(expect, "x-fmt/"+strs[2])
		} else {
			return errors.New("long string encountered: " + path)
		}
		names = append(names, path)
		return nil
	}
	suite := filepath.Join(*testdata, "skeleton-suite")
	_, err = os.Stat(suite)
	if err != nil {
		t.Fatal(err)
	}
	err = filepath.Walk(suite, wf)
	if err != nil {
		t.Fatal(err)
	}
	matches, err := multiIdentifyT(s, suite)
	if err != nil {
		t.Fatal(err)
	}
	if len(expect) != len(matches) {
		t.Error("Expect should equal matches")
	}
	var iter int
	for i, v := range expect {
		if !check(v, matches[i]) {
			t.Errorf("Failed to match signature %v; got %v; expected %v", names[i], matchString(matches[i]), v)

		} else {
			iter++
		}
	}
	if iter != len(expect) {
		t.Errorf("Matched %v out of %v signatures", iter, len(expect))
	}
}

func TestTip(t *testing.T) {
	expect := "fmt/669"
	err := setup()
	if err != nil {
		t.Error(err)
	}
	buf := bytes.NewReader([]byte{0x00, 0x4d, 0x52, 0x4d, 0x00})
	c, _ := s.Identify(buf, "test.mrw", "")
	for _, i := range c {
		if i.String() != expect {
			t.Errorf("First buffer: expecting %s, got %s", expect, i)
		}
	}
	buf = bytes.NewReader([]byte{0x00, 0x4d, 0x52, 0x4d, 0x00})
	c, _ = s.Identify(buf, "test.mrw", "")
	for _, i := range c {
		if i.String() != expect {
			t.Errorf("Second buffer: expecting %s, got %s", expect, i)
		}
	}
	buf = bytes.NewReader([]byte{0x00, 0x4d, 0x52, 0x4d, 0x00})
	c, _ = s.Identify(buf, "test.mrw", "")
	for _, i := range c {
		if i.String() != expect {
			t.Errorf("Third buffer: expecting %s, got %s", expect, i)
		}
	}
}

// TestDROID tests -multi DROID. Samples from https://github.com/richardlehane/siegfried/issues/146
func TestDROID(t *testing.T) {
	if err := setup(config.SetMulti("DROID")); err != nil {
		t.Fatal(err)
	}
	expect1 := []string{"fmt/41", "fmt/96"}
	jpghtml := [60]uint8{
		0xFF, 0xD8, 0xFF, 0x3C, 0x68, 0x74, 0x6D, 0x6C, 0x3E, 0x54, 0x48, 0x49,
		0x53, 0x20, 0x46, 0x49, 0x4C, 0x45, 0x20, 0x53, 0x48, 0x4F, 0x55, 0x4C,
		0x44, 0x20, 0x49, 0x44, 0x45, 0x4E, 0x54, 0x49, 0x46, 0x59, 0x20, 0x41,
		0x53, 0x20, 0x4A, 0x50, 0x45, 0x47, 0x20, 0x41, 0x4E, 0x44, 0x20, 0x48,
		0x54, 0x4D, 0x4C, 0x3C, 0x2F, 0x68, 0x74, 0x6D, 0x6C, 0x3E, 0xFF, 0xD9,
	}
	expect2 := []string{"fmt/41", "x-fmt/384"}
	jpgmov := [69]uint8{
		0xFF, 0xD8, 0xFF, 0x00, 0x6D, 0x6F, 0x6F, 0x76, 0x00, 0x00, 0x00, 0x00,
		0x6D, 0x76, 0x68, 0x64, 0x54, 0x48, 0x49, 0x53, 0x20, 0x46, 0x49, 0x4C,
		0x45, 0x20, 0x53, 0x48, 0x4F, 0x55, 0x4C, 0x44, 0x20, 0x49, 0x44, 0x45,
		0x4E, 0x54, 0x49, 0x46, 0x59, 0x20, 0x41, 0x53, 0x20, 0x51, 0x55, 0x49,
		0x43, 0x4B, 0x54, 0x49, 0x4D, 0x45, 0x20, 0x4D, 0x4F, 0x56, 0x20, 0x41,
		0x4E, 0x44, 0x20, 0x4A, 0x50, 0x45, 0x47, 0xFF, 0xD9,
	}
	buf := bytes.NewReader(jpghtml[:])
	c, _ := s.Identify(buf, "test.jpg", "")
	if len(c) != len(expect1) || (c[0].String() != expect1[0] && c[0].String() != expect1[1]) || (c[1].String() != expect1[0] && c[1].String() != expect1[1]) {
		t.Errorf("-multi DROID: expected %v; got %v", expect1, c)
	}
	buf = bytes.NewReader(jpgmov[:])
	c, _ = s.Identify(buf, "test.jpg", "")
	if len(c) != len(expect2) || (c[0].String() != expect2[0] && c[0].String() != expect2[1]) || (c[1].String() != expect2[0] && c[1].String() != expect2[1]) {
		t.Errorf("-multi DROID: expected %v; got %v", expect2, c)
	}
	setup(config.Clear())
}

func Test363(t *testing.T) {
	repetitions := 10000
	iter := 0
	expect := "fmt/363"
	err := setup()
	if err != nil {
		t.Error(err)
	}
	segy := func(l int) []byte {
		b := make([]byte, l)
		for i := range b {
			if i > 21 {
				break
			}
			b[i] = 64
		}
		copy(b[l-9:], []byte{01, 00, 00, 00, 01, 00, 00, 01, 00})
		return b
	}
	se := segy(3226)
	for i := 0; i < repetitions; i++ {
		buf := bytes.NewReader(se)
		c, _ := s.Identify(buf, "test.seg", "")
		for _, i := range c {
			iter++
			if i.String() != expect {
				t.Errorf("first buffer on %d iteration: expecting %s, got %s", iter, expect, i)
			}
		}
	}
	iter = 0
	se = segy(3626)
	for i := 0; i < repetitions; i++ {
		buf := bytes.NewReader(se)
		c, _ := s.Identify(buf, "test2.seg", "")
		for _, i := range c {
			iter++
			if i.String() != expect {
				t.Errorf("Second buffer on %d iteration: expecting %s, got %s", iter, expect, i)
			}
		}
	}
}

// Benchmarks
func benchidentify(ext string) {
	setup()
	file := filepath.Join(*testdata, "benchmark", "Benchmark")
	file += "." + ext
	identifyT(s, file)
}

func BenchmarkACCDB(bench *testing.B) {
	for i := 0; i < bench.N; i++ {
		benchidentify("accdb")
	}
}

func BenchmarkBMP(bench *testing.B) {
	for i := 0; i < bench.N; i++ {
		benchidentify("bmp")
	}
}

func BenchmarkDOCX(bench *testing.B) {
	for i := 0; i < bench.N; i++ {
		benchidentify("docx")
	}
}

func BenchmarkGIF(bench *testing.B) {
	for i := 0; i < bench.N; i++ {
		benchidentify("gif")
	}
}

func BenchmarkJPG(bench *testing.B) {
	for i := 0; i < bench.N; i++ {
		benchidentify("jpg")
	}
}

func BenchmarkMSG(bench *testing.B) {
	for i := 0; i < bench.N; i++ {
		benchidentify("msg")
	}
}

func BenchmarkODT(bench *testing.B) {
	for i := 0; i < bench.N; i++ {
		benchidentify("odt")
	}
}

func BenchmarkPDF(bench *testing.B) {
	for i := 0; i < bench.N; i++ {
		benchidentify("pdf")
	}
}

func BenchmarkPNG(bench *testing.B) {
	for i := 0; i < bench.N; i++ {
		benchidentify("png")
	}
}

func BenchmarkPPTX(bench *testing.B) {
	for i := 0; i < bench.N; i++ {
		benchidentify("pptx")
	}
}

func BenchmarkRTF(bench *testing.B) {
	for i := 0; i < bench.N; i++ {
		benchidentify("rtf")
	}
}

func BenchmarkTIF(bench *testing.B) {
	for i := 0; i < bench.N; i++ {
		benchidentify("tif")
	}
}

func BenchmarkXLSX(bench *testing.B) {
	for i := 0; i < bench.N; i++ {
		benchidentify("xlsx")
	}
}

func BenchmarkXML(bench *testing.B) {
	for i := 0; i < bench.N; i++ {
		benchidentify("xml")
	}
}

func BenchmarkMulti(bench *testing.B) {
	dir := filepath.Join(*testdata, "benchmark")
	for i := 0; i < bench.N; i++ {
		multiIdentifyT(s, dir)
	}
}

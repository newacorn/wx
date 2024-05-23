package main

import (
	"bytes"
	crand "crypto/rand"
	"encoding/gob"
	"io"
	"math/rand"
	"testing"
	"time"
	_ "unsafe"

	"fasthttp-routing/middleware/session"
	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"helpers/unsafefn"
	"noescape"
)

var reader io.Reader

func init() {
	reader = &simpleReader{}
	if rand.Intn(4) > 9 {
		reader = crand.Reader
	}
}
func benchMarshalMap(b *testing.B) {
	m := make(map[string]interface{})
	m["abc"] = "123344"
	m["defg"] = "aflsdfjlksdjflksjdfljsdlf"
	m["login"] = "1123239fjsaldfjlsdaafsdfsdfsdf"
	m["lgin"] = "3121289182798127"
	m["122323"] = "322222222222222222aflsfjlasjflasjlfjasldfjlsadfjlas23293890283lajflasfsdf"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b2 := bytes.Buffer{}
		g := gob.NewEncoder(&b2)
		err := g.Encode(m)
		if err != nil {
			panic(err)
		}
	}
	/*
		ast.Node{}
		ast.NewString("11232")
		ast.NewObject([]ast.Pair)
	*/
}
func BenchmarkWithGob(b *testing.B) {
	benchMarshalMap(b)
}
func BenchmarkWithSonic(b *testing.B) {
	benchMarshalAstNode(b)
}
func BenchmarkWithSonicMap(b *testing.B) {
	benchMarshalMap2(b)
}
func BenchmarkWithSonicMapNode(b *testing.B) {
	benchMarshalMap3(b)
}
func benchMarshalMap2(b *testing.B) {
	m := make(map[string]interface{})
	m["abc"] = "123344"
	m["defg"] = "aflsdfjlksdjflksjdfljsdlf"
	m["login"] = "1123239fjsaldfjlsdaafsdfsdfsdf"
	m["lgin"] = "3121289182798127"
	m["122323"] = "322222222222222222aflsfjlasjflasjlfjasldfjlsadfjlas23293890283lajflasfsdf"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := sonic.Marshal(m)
		if err != nil {
			panic(err)
		}
	}
}
func benchMarshalMap3(b *testing.B) {
	m := make(map[string]ast.Node)
	m["abc"] = ast.NewString("123344")
	m["defg"] = ast.NewString("aflsdfjlksdjflksjdfljsdlf")
	m["login"] = ast.NewString("1123239fjsaldfjlsdaafsdfsdfsdf")
	m["lgin"] = ast.NewString("3121289182798127")
	m["122323"] = ast.NewString("322222222222222222aflsfjlasjflasjlfjasldfjlsadfjlas23293890283lajflasfsdf")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := sonic.Marshal(m)
		if err != nil {
			panic(err)
		}
	}
}

func benchMarshalAstNode(b *testing.B) {
	obj := ast.NewObject([]ast.Pair{{"abc", ast.NewString("123344")}, {"defg", ast.NewString("aflsdfjlksdjflksjdfljsdlf")}, {
		"login", ast.NewString("1123239fjsaldfjlsdaafsdfsdfsdf")}, {
		"lgin", ast.NewNumber("3121289182798127")}, {
		"122323", ast.NewString("322222222222222222aflsfjlasjflasjlfjasldfjlsadfjlas23293890283lajflasfsdf"),
	},
	})

	// obj.UnmarshalJSON()
	// d := []byte{'1'}
	// g
	// n1.UnmarshalJSON()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r, err := obj.MarshalJSON()

		// n := ast.Node{}
		ss, _ := sonic.Get(r)
		r2 := ss.Get("122323")
		_ = r2
		n9 := ast.Node{}
		_ = n9.UnmarshalJSON(r)
		n9.LoadAll()
		n10 := n9.Get("122323")
		_ = n10
		// sonic.Unmarshal(r, &n)
		if err != nil {
			println(string(r))
			panic(err)
		}
	}

}
func BenchmarkMarshTime(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		t := time.Now()
		var t2 time.Time
		r, _ := sonic.Marshal(t)
		err := sonic.Unmarshal(r, &t2)
		if !t2.Equal(t) {
			panic("error")
		}
		if err != nil {
			panic(err)
		}
	}
}
func BenchmarkMarshTime2(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		t := time.Now()
		var n int64
		n = t.UTC().UnixNano()
		r, _ := sonic.Marshal(n)
		err := sonic.Unmarshal(r, &n)

		tr := time.Unix(0, n)
		_ = tr
		if !tr.Equal(t) {
			panic("dont equal")
		}
		if err != nil {
			panic(err)
		}
	}
}
func BenchmarkInterface(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b := make([]byte, 40)
		// _, err := (&simpleReader{}).Read(b)
		_, err := reader.Read(b)
		// _, err := rand.Read(b)
		if err != nil {
			panic(err)
		}

	}
}
func BenchmarkNoescape2(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b := make([]byte, 40)
		// _, err := noescape.Read(rand.Reader, b)
		_, err := noescape.Read(&simpleReader{}, b)
		if err != nil {
			panic(err)
		}

	}
	session.New()
}
func BenchmarkNoescape(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b := make([]byte, 40)
		// _, err := noescape.Read(rand.Reader, b)
		_, err := unsafefn.NoescapeRead(&simpleReader{}, b)
		if err != nil {
			panic(err)
		}

	}
	session.New()
}

//go:linkname noescapeRead fasthttp-routing/middleware/session.Helper
//go:noescape
func noescapeRead(r io.Reader, b []byte) (int, error)

type simpleReader struct {
}

func (s *simpleReader) Read(p []byte) (int, error) {
	str := "1111111111112222222222222222222222222244444444444444444444445555555555"
	return copy(p, str), nil
}

package main

import (
	"bytes"
	"encoding/gob"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"helpers/unsafefn"
)

var str = `{"good1":"wafdsfffffffasdafsafwonisdfd","cafdsfdf":"aaaaaaaaaaaaaaaaannnnnd","old":{"key1":"afsdfasfoweir2323werw0e8rwerwerfsdfb","key2":"af982032093-23reiwfsiafjsdfsd"},"new":{"key4":"123fsdfasfoweir2323werw0e8rwerwerfsdfb","key332":"af982032093-23reiwfsiafjsdfsd"}}`
var gobBytes []byte

func init() {
	f, err := os.Open("/Users/acorn/workspace/programming/golang/echostudy/fasthttp-routing/middleware/session/main/out.txt")
	defer func() { _ = f.Close() }()
	if err != nil {
		panic(err)
	}
	gobBytes, err = io.ReadAll(f)
	if err != nil {
		panic(err)
	}
}

func BenchmarkNode(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// n, err := sonic.Get(unsafefn.StoB(str))
		n, err := sonic.GetFromString(str)
		n.LoadAll()
		if err != nil {
			panic(err)
		}
		r, err := n.Get("old").Get("key1").String()
		r, err = n.Get("cafdsfdf").String()
		r, err = n.Get("good1").String()
		r, err = n.Get("new").Get("key4").String()
		r, err = n.Get("new").Get("key332").String()
		if err != nil {
			println(err)
		}
		if r == "" {
			panic("err")
		}
		r2, err := n.MarshalJSONBuf()
		if err != nil {
			panic(err)
		}
		ast.FreeBuffer(r2)
		ast.ReleaseNodes(&n)

	}
}

type Q1 map[string]interface{}

func BenchmarkGob(b *testing.B) {
	gob.Register(map[string]interface{}{})

	s := strings.NewReader(unsafefn.BtoS(gobBytes))
	b2 := bytes.Buffer{}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var q Q1
		err := gob.NewDecoder(s).Decode(&q)
		if err != nil {
			panic(err)
		}
		s.Seek(0, 0)
		r1 := q["old"]
		r2 := r1.(map[string]interface{})
		r3 := r2["key1"]
		if r3 == "" {
			panic("b")
		}
		// b2 := bytes.Buffer{}
		gob.NewEncoder(&b2).Encode(q)
		b2.Reset()

	}
}

func BenchmarkMap(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// var r map[string]interface{}
		var r Q1
		err := sonic.UnmarshalString(str, &r)
		if err != nil {
			panic(err)
		}
		/*
			f, err := os.Create("out.txt")
			if err != nil {
				panic(err)
			}
				err = gob.NewEncoder(f).Encode(r)
				if err != nil {
					panic(err)
				}
				_ = f.Close()
		*/
		/*
			r, err := n.Get("old").Get("key1").String()
			r, err = n.Get("cafdsfdf").String()
			r, err = n.Get("good1").String()
			r, err = n.Get("new").Get("key4").String()
			r, err = n.Get("new").Get("key332").String()
		*/
		r3 := r["cafdsfdf"]
		r4 := r3.(string)
		_ = r4
		r3 = r["good1"]
		r4 = r3.(string)
		_ = r4
		r3 = r["new"]
		r5 := r3.(map[string]interface{})
		r6 := r5["key4"].(string)
		_ = r6
		_ = r4
		r3 = r["new"]
		r5 = r3.(map[string]interface{})
		r6 = r5["key332"].(string)
		_ = r6

		r1 := r["old"]
		r2, ok := r1.(map[string]interface{})
		if !ok {
			panic("23")
		}
		s1 := r2["key1"]
		s2 := s1.(string)
		if s2 == "" {
			panic("b")
		}
		sonic.Marshal(r)

		r11 := r["old"]
		r22, ok := r11.(map[string]interface{})
		if !ok {
			panic("23")
		}
		s11 := r22["key1"]
		s22 := s11.(string)
		if s22 == "" {
			panic("b")
		}
		sonic.Marshal(r)
	}
}

// var str = `{"good1":"wafdsfffffffasdafsafwonisdfd","cafdsfdf":"aaaaaaaaaaaaaaaaannnnnd","old":{"key1":"afsdfasfoweir2323werw0e8rwerwerfsdfb","key2":"af982032093-23reiwfsiafjsdfsd"},"new":{"key4":"123fsdfasfoweir2323werw0e8rwerwerfsdfb","key332":"af982032093-23reiwfsiafjsdfsd"}}`

func BenchmarkMapData(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		data := Data2{values: map[string]interface{}{}}
		data.Put("good1", "wafdsfffffffasdafsafwonisdfd")
		data.Put("cafdsfdf", "aaaaaaaaaaaaaaaaannnnnd")
		data.SetByPath("afsdfasfoweir2323werw0e8rwerwerfsdfb", "old", "key1")
		data.SetByPath("af982032093-23reiwfsiafjsdfsd", "old", "key2")
		data.SetByPath("123fsdfasfoweir2323werw0e8rwerwerfsdfb", "new", "key4")
		data.SetByPath("af982032093-23reiwfsiafjsdfsd", "new", "key332")
		encodedData, err := sonic.Marshal(data.values)
		if err != nil {
			panic(err)
		}
		var p map[string]interface{}
		err = sonic.Unmarshal(encodedData, &p)
		if err != nil {
			panic(err)
		}
		data.values = p
		ok, r := data.GetByPath("old", "key1")
		if !ok {
			panic("fetch error.")
		}
		if r.(string) != "afsdfasfoweir2323werw0e8rwerwerfsdfb" {
			panic("fetch error.")
		}
	}
}
func BenchmarkNodeData(b *testing.B) {
	// ast.quote2 = ast.quote
	b.ResetTimer()
	b.ReportAllocs()
	ast.C.Store(0)
	ast.C2.Store(0)
	for i := 0; i < b.N; i++ {
		data := Data{ast.NewObject(nil)}
		data.Put2("good1", "wafdsfffffffasdafsafwonisdfd")
		data.Put2("cafdsfdf", "aaaaaaaaaaaaaaaaannnnnd")
		// data.SetByPath2("afsdfasfoweir2323werw0e8rwerwerfsdfb", "old", "key1")
		data.SetByPath2("afsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfb8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfb8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfb8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfbafsdfasfoweir2323werw0e8rwerwerfsdfb", "old", "key1")
		data.SetByPath2("af982032093-23reiwfsiafjsdfsd", "old", "key2")
		data.SetByPath2("123fsdfasfoweir2323werw0e8rwerwerfsdfb", "new", "key4")
		data.SetByPath2("af982032093-23reiwfsiafjsdfsd", "new", "key332")

		// pp := ast.NewObject([]ast.Pair{{"key", ast.NewString("12323")}})
		// qq, _ := pp.MarshalJSON()

		ast.UpdateNodeLen(&data.Node)
		bufBytes, err := data.MarshalJSONBuf()
		if err != nil {
			panic(err)
		}
		ast.ReleaseNodes(&data.Node)
		n, err := sonic.GetFromString(unsafefn.BtoS(*bufBytes))
		if err != nil {
			panic(err)
		}
		err = n.LoadAll()
		if err != nil {
			panic(err)
		}
		ok, r := n.GetByPath2("old", "key2")
		if !ok {
			panic("fetch error")
		}
		s, err := r.String()
		if err != nil || s != "af982032093-23reiwfsiafjsdfsd" {
			panic("dont equal.")
		}
		// ast.FreeBuffer(bufBytes)
		ast.FreeBuffer(bufBytes)
		ast.ReleaseNodes(&n)
	}
}
func BenchmarkNodeDataAppend(b *testing.B) {
	// ast.quote2 = quote
	b.ResetTimer()
	b.ReportAllocs()
	ast.C.Store(0)
	ast.C2.Store(0)
	for i := 0; i < b.N; i++ {
		data := Data{ast.NewObject(nil)}
		data.Put2("good1", "wafdsfffffffasdafsafwonisdfd")
		data.Put2("cafdsfdf", "aaaaaaaaaaaaaaaaannnnnd")
		data.SetByPath2("afsdfasfoweir2323werw0e8rwerwerfsdfb", "old", "key1")
		data.SetByPath2("af982032093-23reiwfsiafjsdfsd", "old", "key2")
		data.SetByPath2("123fsdfasfoweir2323werw0e8rwerwerfsdfb", "new", "key4")
		data.SetByPath2("af982032093-23reiwfsiafjsdfsd", "new", "key332")

		// pp := ast.NewObject([]ast.Pair{{"key", ast.NewString("12323")}})
		// qq, _ := pp.MarshalJSON()

		ast.UpdateNodeLen(&data.Node)
		bufBytes, err := data.MarshalJSONBuf()
		if err != nil {
			panic(err)
		}
		ast.ReleaseNodes(&data.Node)
		n, err := sonic.GetFromString(unsafefn.BtoS(*bufBytes))
		if err != nil {
			panic(err)
		}
		err = n.LoadAll()
		if err != nil {
			panic(err)
		}
		ok, r := n.GetByPath2("old", "key1")
		if !ok {
			panic("fetch error")
		}
		s, err := r.String()
		if err != nil || s != "afsdfasfoweir2323werw0e8rwerwerfsdfb" {
			panic("dont equal.")
		}
		// ast.FreeBuffer(bufBytes)
		ast.FreeBuffer(bufBytes)
		ast.ReleaseNodes(&n)
	}
}
func Quote(buf *[]byte, v string) {
	*buf = append(*buf, '"')
	*buf = append(*buf, v...)
	*buf = append(*buf, '"')
}

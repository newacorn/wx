package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/redis/rueidis"
)

func main9() {
	cli, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{"127.0.0.1:6379"}, ForceSingleClient: true})
	if err != nil {
		panic(err)
	}
	result := cli.Do(context.Background(), cli.B().Expireat().Key("223").Timestamp(time.Now().Add(time.Second*1000).Unix()).Build())
	fmt.Println(result.AsInt64())
	cli.Close()
}

type ByteArr []byte
type Args []ArgKv
type ArgKv struct {
	K ByteArr
	V ByteArr
}

//	func (b ByteArr) MarshalJSON() ([]byte, error) {
//		return []byte("\"" + string(b) + "\""), nil
//	}
type Q map[string]interface{}
type S struct {
	A string
	B int
	C map[string]interface{}
}

func main() {
	var str = `{"good1":"wafdsfffffffasdafsafwonisdfd","cafdsfdf":"aaaaaaaaaaaaaaaaannnnnd","old":{"key1":"afsdfasfoweir2323werw0e8rwerwerfsdfb","key2":"af982032093-23reiwfsiafjsdfsd"},"new":{"key4":"123fsdfasfoweir2323werw0e8rwerwerfsdfb","key332":"af982032093-23reiwfsiafjsdfsd"}}`
	_ = str
	m := make(map[string]interface{})
	m["a"] = make(map[string]interface{})
	v := m["a"]
	r := v.(map[string]interface{})
	r["c"] = 999
	clear(m)
	println(&m)
	type C struct {
		C int
		B bool
	}
	type M struct {
		A string
		Q C
	}
	arr := []M{M{A: "a", Q: C{C: 999, B: true}}, M{A: "b"}}
	clear(arr[:])
	println(9)
	// NodeData()
	// println("alloc", ast.C2.Load())
	// println("relase", ast.C.Load())

}
func NodeData() {
	data := &Data{ast.NewObject(nil)}
	data.Put2("a", "bbbb")
	data.SetByPath2("a error message.", "old", "a")
	ok, n := data.Get2("a")
	ok, n = data.GetByPath2("old", "a")
	if ok {
		r1, err := n.String()
		if err == nil {
			if r1 != "a error message." {
				panic("")
			}
		}
	}
	ast.ReleaseNodes(&data.Node)
}
func MapData() {

	// ast.ReleaseNodes(&data.Node)
}

func main11() {
	n := ast.NewString(strings.Repeat("1", 2048))
	r, err := n.MarshalJSON()
	if err != nil {
		panic(err)
	}
	println(len(r))
}
func main222() {
	gob.Register(map[string]interface{}{})
	gob.Register(map[string]string{})
	// gob.Register(S{})

	var b Q = make(map[string]interface{})
	b["abc"] = "232323"
	b["key2"] = "99929fasdfsfd232323aw0er9u203u9r023ru203u9rf023"
	b["key3"] = "99929fasdfsfd232323f2w03ruwoeirweiorupqwoieuropiwefjuopw"
	b["key4"] = "99929fasdfsfd232323afsf0w3920392u30"
	b["key5"] = "111111111fasdflasdfsdfsdafs99929fasdfsfd232323"
	b["old"] = map[string]string{"afsdf": "232323", "23i2323234234": "222222211111111"}
	b["new"] = map[string]string{"12323afsdf": "777232323", "999923i2323234234": "22222221111111000001"}
	// b["old"] = S{}
	b1 := bytes.Buffer{}
	f, err := os.Create("out.txt")
	if err != nil {
		panic(err)
	}
	err = gob.NewEncoder(&b1).Encode(b)
	if err != nil {
		panic(err)
	}
	println(b1.String())
	f.Close()
}
func main10() {
	o := ast.NewObject([]ast.Pair{{"a", ast.NewString("b")}, {"c", ast.NewString("d")},
		{"old", ast.NewObject([]ast.Pair{{"a", ast.NewString("b")}, {"c", ast.NewString("d")}})},
	})
	r, err := o.MarshalJSON()
	if err != nil {
		panic(err)
	}
	println(string(r))
	n, err := sonic.Get(r)
	if err != nil {
		panic(err)
	}
	n.LoadAll()
	rn := n.GetByPath("old", "a")
	r2, err := rn.String()
	println(r2)
	/*
		s := Args{{K: []byte("a"), V: []byte("b")}, {}}
		r, err := sonic.MarshalString(s)
		if err != nil {
			panic(err)
		}
		println(r)

	*/
}

func main99() {
	b := []byte("12323233")
	dst := make([]byte, base64.RawURLEncoding.EncodedLen(len(b)))
	copy(dst, b)
	base64.RawURLEncoding.Encode(dst, b)
	println(string(dst))

	t1 := time.Now().UTC()
	r := t1.Nanosecond()
	// t1.UnixNano()

	println(t1.String())
	t := time.Unix(0, int64(r))
	println(t.String())
	// l := base64.RawURLEncoding.DecodedLen(len(dst))
	// dst4 := make([]byte, len(dst2))
	n, _ := base64.RawURLEncoding.Decode(dst, dst)
	println(string(dst[:n]), "**")
}

type Data struct {
	ast.Node
}

func (d *Data) Get(key string) (n *ast.Node) {
	return d.Node.Get(key)
}
func (d *Data2) Get(key string) (n interface{}) {
	return d.values[key]
}
func (d *Data2) Put(key string, value string) {
	d.values[key] = value
}
func (d *Data2) Delete(key string) {
	delete(d.values, key)
}
func (d *Data) Put(key string, value string) (ok bool, err error) {
	ok, err = d.Node.Set(key, ast.NewString(value))
	return
}
func (d *Data2) GetByPath(path ...string) (ok bool, r interface{}) {
	m := d.values
	l := len(path)
	for i := range path {
		last := l-1 == i
		r1 := m[path[i]]
		if r1 == nil {
			if last {
				ok = true
				r = r1
			}
			return
		}
		r2, ok1 := r1.(map[string]interface{})
		if last {
			r = r1
			ok = true
			return
		}
		if ok1 {
			m = r2
			continue
		} else {
			return
		}

	}
	return
}
func (d *Data) Delete(key string) (okk bool, err error) {
	return d.Node.Unset(key)
}
func (d *Data) GetByPath(path ...string) (ok bool, r *ast.Node) {
	// d.Node.Get("")
	n := &d.Node
	l := len(path)
	var last bool
	for i := range path {
		last = l-1 == i
		n = n.Get(path[i])
		if !n.Exists() {
			return
		}
		if last {
			r = n
			ok = true
			return
		}
	}
	return
}

func (d *Data) SetByPath(v string, path ...string) (err error) {
	var root = d
	if root.Node.Type() != ast.V_OBJECT {
		panic("root not object")
	}
	var parent *ast.Node
	var child *ast.Node
	parent = &d.Node
	l := len(path)
	for i := range path {
		child = parent.Get(path[i])
		if !child.Exists() {
			if i == l-1 {
				parent.Set(path[i], ast.NewString(v))
			} else {
				parent.Set(path[i], ast.NewObject(nil))
				parent = parent.Get(path[i])
			}
		} else if child.Type() == ast.V_OBJECT {
			if i == l-1 {
				child.Set(path[i], ast.NewString(v))
			} else {
				tmp := parent.Get(path[i])
				if tmp.Exists() {
					parent = tmp
				} else {
					parent.Set(path[i], ast.NewObject(nil))
					parent = parent.Get(path[i])
				}
			}
		} else {
			return errors.New("cannot set value by path")
		}
	}
	return
}

type Data2 struct {
	values map[string]interface{}
}

func (d Data2) SetByPath(v string, path ...string) (err error) {
	var parent map[string]interface{}
	// var child map[string]interface{}
	var l = len(path)
	parent = d.values
	for i := range path {
		c1 := parent[path[i]]
		if c1 == nil {
			if i == l-1 {
				parent[path[i]] = v
				return
			} else {
				c1 := make(map[string]interface{})
				parent[path[i]] = c1
				parent = c1
				continue
			}
		} else if c2, ok := c1.(map[string]interface{}); ok {
			if i == l-1 {
				parent[path[i]] = v
				return
			} else {
				parent = c2
				/*
					c3 := c2[path[i]]
					if c3 == nil {
						c4 := make(map[string]interface{})
						c2[path[i]] = c4
						parent = c4
					} else {
						if c4, ok := c3.(map[string]interface{}); ok {
							parent = c4
						} else {
							err = errors.New("type error")
							return
						}
					}
				*/
			}
		} else {
			if i == l-1 {
				parent[path[i]] = v
				return
			}
			c := make(map[string]interface{})
			parent[path[i]] = c
			parent = c
			continue
		}
	}
	return
}

package main

import (
	"encoding/xml"
	"log"
)

func main_() {
	type A struct {
		Name    string `xml:",chardata"`
		Age     int    `xml:",cdata""`
		Comment string `xml:",comment"`
	}
	a := A{"name", 1, "comment"}
	r, err := xml.Marshal(a)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(r))

	str2 := `<A>name<![CDATA[1]]><!--comment--></A>`

	type A1 struct {
		XMLName xml.Name `xml:"A"`
		Age     int      `xml:",cdata"`
		Name    string   `xml:",chardata"`
		Comment string   `xml:",comment"`
	}
	var a1 A1
	err = xml.Unmarshal([]byte(str2), &a1)
	log.Fatal(a1, err)
}

var x = `<xml>
 <ToUserName><![CDATA[粉丝号]]></ToUserName>
 <FromUserName><![CDATA[公众号]]></FromUserName>
 <CreateTime>1460541339</CreateTime>
 <MsgType><![CDATA[text]]></MsgType>
 <Content><![CDATA[test]]></Content>
</xml>`

func main() {
	type B struct {
		CDATA string `xml:",cdata"`
	}
	type A struct {
		XMLName      xml.Name `xml:"xml"`
		ToUserName   B        `xml:"ToUserName"`
		FromUserName string   `xml:"FromUserName"`
		CreateTime   int64    `xml:"CreateTime"`
		MsgType      string   `xml:"MsgType"`
		Content      string   `xml:"Content"`
	}
	var a A
	err := xml.Unmarshal([]byte(x), &a)
	log.Println(err, a)
	// log.Fatal(a, err)
	// 2024/05/19 15:35:55 {{ xml} 粉丝号 公众号 1460541339 text test}
	r, _ := xml.Marshal(a)
	log.Println(string(r))

}

package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"log"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	routing "fasthttp-routing"
	"github.com/newacorn/fasthttp"
	"helpers/unsafefn"
)

var AuthServerToken = []byte("001991acorn")
var SECRET = "cb4bcbff2b89c1a345c78d41711afc2d"
var APPID = "wxd16970b7664562ed"
var WxToken atomic.Value
var WxTokenRequestUrl = "https://api.weixin.qq.com/cgi-bin/stable_token"
var GrantType = "client_credential"

type stableTokenCredential struct {
	AppID     string `json:"appid"`
	Secret    string `json:"secret"`
	GrantType string `json:"grant_type"`
}
type WxTokenResponseOk struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}
type WxTokenResponseError struct {
	ErrorCode int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
}

var tokenReady = make(chan struct{})

func init() {
	go updateToken()
}
func main() {
	<-tokenReady
	r := routing.New()
	r.Get("/wx", handleSerVerify)
	r.Post("/wx", copyUserMessage)
	r.Get("/token", GetToken)

	r.Post("/", func(ctx *routing.Ctx) error {
		log.Println(ctx.Request.URI().String())
		log.Println(string(ctx.Request.Body()))
		_, _ = ctx.WriteString("success")
		return nil
	})
	server := fasthttp.Server{Handler: r.HandleRequest}
	log.Fatal(server.ListenAndServe(":80"))
}
func handleSerVerify(ctx *routing.Ctx) (err error) {
	args := ctx.URI().QueryArgs()
	signature := args.Peek("signature")
	if signature == nil {
		return
	}
	timestamp := args.Peek("timestamp")
	nonce := args.Peek("nonce")
	echostr := args.Peek("echostr")
	list := [][]byte{AuthServerToken, timestamp, nonce}
	sort.Slice(list, func(i, j int) bool {
		return string(list[i]) < string(list[j])
	})
	data := bytes.Join(list, nil)
	t := sha1.Sum(data)
	result := hex.EncodeToString(t[:])
	if bytes.Equal(unsafefn.StoB(result), signature) {
		_, _ = ctx.Write(echostr)
	}
	return nil
}
func GetToken(ctx *routing.Ctx) (err error) {
	_, _ = ctx.WriteString(WxToken.Load().(string))
	return
}
func getAppId() (appid string, err error) {
	return APPID, nil
}
func getSecret() (secret string, err error) {
	return SECRET, nil
}
func getWxTokenRequestUrl() (url string, err error) {
	return WxTokenRequestUrl, nil
}
func requestWxToken() (token string, err error) {
	var appId, secret string
	appId, err = getAppId()
	if err != nil {
		return
	}
	secret, err = getSecret()
	if err != nil {
		return
	}
	var s = stableTokenCredential{GrantType: GrantType, AppID: appId, Secret: secret}
	var reqBody []byte
	reqBody, err = json.Marshal(s)
	if err != nil {
		return
	}
	wxTokenUrl, err := getWxTokenRequestUrl()
	if err != nil {
		return
	}
	cli := fasthttp.Client{}
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()
	req.Header.SetMethod("POST")
	req.SetRequestURI(wxTokenUrl)
	req.SetBody(reqBody)
	err = cli.Do(req, resp)
	if err != nil {
		return
	}
	statusCode := resp.StatusCode()
	if statusCode != 200 {
		err = errors.New("request wx token failed: url " + wxTokenUrl + " status code: " + strconv.Itoa(statusCode))
		return
	}
	respBody := resp.Body()
	if bytes.Contains(respBody, []byte("errcode")) {
		err = errors.New(string(respBody))
		return
	}
	var ok WxTokenResponseOk
	err = json.Unmarshal(respBody, &ok)
	if err != nil {
		return
	}
	return ok.AccessToken, nil
}

func updateToken() {
	t := time.Timer{}
	time.NewTimer(time.Second * 270)
	for {
		token, err := requestWxToken()
		if err != nil {
			// WxToken.Store("")
			log.Println(err)
		} else {
			WxToken.Store(token)
			tokenReady <- struct{}{}
		}
		<-t.C
	}
}

// var wxTokenResponse WxTokenResponse
func getWxToken() string {
	return WxToken.Load().(string)
}

func copyUserMessage(ctx *routing.Ctx) (err error) {
	type ToUserName struct {
		CDATA string `xml:",cdata"`
	}
	type FromUserName = ToUserName
	type MsgType = ToUserName
	type Content = ToUserName
	type PicUrl = ToUserName
	type MediaId = ToUserName
	type MsgID = ToUserName
	type commonMessage struct {
		ToUserName   ToUserName
		FromUserName FromUserName
		CreateTime   int64
		MsgType      MsgType
	}
	type imageMessage struct {
		XMLName xml.Name `xml:"xml"`
		commonMessage
		PicUrl  PicUrl
		MediaId MediaId
		MsgID   MsgID
	}
	type textMessage struct {
		XMLName xml.Name `xml:"xml"`
		commonMessage
		Content Content
	}
	reqBody := ctx.Request.Body()
	var n textMessage
	err = xml.Unmarshal(reqBody, &n)
	if err != nil {
		_, _ = ctx.WriteString("success")
		return
	}
	n.FromUserName, n.ToUserName = n.ToUserName, n.FromUserName
	if n.MsgType.CDATA == "text" {

		respBody, err := xml.Marshal(n)
		if err != nil {
			log.Println(err)
			_, _ = ctx.WriteString("success")
			return err
		}
		log.Println(string(respBody))
		_, _ = ctx.Write(respBody)
		return err
	}
	if n.MsgType.CDATA == "image" {
		var n1 imageMessage
		log.Println(string(reqBody))
		err := xml.Unmarshal(reqBody, &n1)
		if err != nil {
			log.Println(err)
			_, _ = ctx.WriteString("success")
			return nil
		}
		type MediaId = ToUserName
		type imageResp struct {
			XMLName xml.Name `xml:"xml"`
			commonMessage
			MediaId MediaId `xml:"Image>MediaId"`
		}
		var n2 imageResp
		// n2.FromUserName = n.FromUserName
		n2.commonMessage = n.commonMessage
		n2.MediaId = n1.MediaId
		resp, err := xml.Marshal(n2)
		if err != nil {
			log.Println(err)
			_, _ = ctx.WriteString("success")
			return nil
		}
		log.Print(string(resp))
		_, _ = ctx.Write(resp)
		return nil
	}
	// log.
	log.Println(string(reqBody))
	_, _ = ctx.WriteString("success")
	return nil
}

// text resp
/*
<xml>
 <ToUserName><![CDATA[粉丝号]]></ToUserName>
 <FromUserName><![CDATA[公众号]]></FromUserName>
 <CreateTime>1460541339</CreateTime>
 <MsgType><![CDATA[text]]></MsgType>
 <Content><![CDATA[test]]></Content>
</xml>
*/
// image resp
/*
<xml><ToUserName><![CDATA[oXK7P6ZXZCHyMCvqXWSO4Sa4z8YU]]></ToUserName><FromUserName><![CDATA[gh_5a7911974ac4]]></FromUserName><CreateTime>1716115647</CreateTime><MsgType><![CDATA[text]]></MsgType><Content><![CDATA[对方]]></Content></xml>
2024/05/19 18:47:40 <xml><ToUserName><![CDATA[gh_5a7911974ac4]]></ToUserName>
<FromUserName><![CDATA[oXK7P6ZXZCHyMCvqXWSO4Sa4z8YU]]></FromUserName>
<CreateTime>1716115660</CreateTime>
<MsgType><![CDATA[image]]></MsgType>
<PicUrl><![CDATA[http://mmbiz.qpic.cn/mmbiz_jpg/0zymHJFr4D8JacHx0e2BdicByYUMicxaMaiazK2PmltzxCLvhliaeA5Kru9ccDzYGIccvY0pH3aNqmibn1GS7ibZe7qw/0]]></PicUrl>
<MsgId>24568870216193037</MsgId>
<MediaId><![CDATA[8XLnmtfCD3i1CUnicpfKimc_fF1K3GFfE89y-LkRfO9CmNBYlt7gbFz3dHX7mVlv]]></MediaId>
</xml>
*/
// image resp
/*
<xml><ToUserName><![CDATA[oXK7P6ZXZCHyMCvqXWSO4Sa4z8YU]]></ToUserName><FromUserName><![CDATA[gh_5a7911974ac4]]></FromUserName><CreateTime>1716117546</CreateTime><MsgType><![CDATA[text]]></MsgType><Content><![CDATA[123]]></Content></xml>
*/

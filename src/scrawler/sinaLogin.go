package scrawler

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/url"
	"os"
  "encoding/json"
  "bufio"
  "strconv"

	"bytes"
"io"
"time"
)

var header = map[string]string{
  "Host":                      "login.sina.com.cn",
  "Proxy-Connection":          "keep-alive",
  "Cache-Control":             "max-age=0",
  "Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
  "Origin":                    "http://weibo.com",
  "Upgrade-Insecure-Requests": "1",
  "User-Agent":                "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/50.0.2661.94 Safari/537.36",
  "Referer":                   "http://weibo.com",
  "Accept-Language":           "zh-CN,zh;q=0.8,en;q=0.6,ja;q=0.4",
  "Content-Type":              "application/x-www-form-urlencoded",
}

func WeiboLogin(username, passwd string) string{
  //get cookie for sina website
  strCookies := getCookies()
  // crypto username for logining
  su := url.QueryEscape(username)
  su = base64.StdEncoding.EncodeToString([]byte(su))

  // crypto password for logining
  loginInfo := getPreLogin(su)
  sp := encryptPassword(loginInfo, passwd)

  // is need cgi or not
  var cgi string
  if loginInfo["showpin"].(float64) == 1 {
		saveCaptcha(loginInfo["pcid"].(string), strCookies)
		reader := bufio.NewReader(os.Stdin)
		//for {
		fmt.Println("waiting for input captcha...")
		data, _, _ := reader.ReadLine()
		cgi = string(data)
  }
  // Do login POST
  loginUrl := `http://login.sina.com.cn/sso/login.php?client=ssologin.js(v1.4.18)`
  // form data params
  strParams := buildParems(su, sp, cgi, loginInfo)
  //_, loginCookies := DoRequest(`POST`, loginUrl, strParams, strCookies, ``, header)
  loginResp, loginCookies := DoRequest(`POST`, loginUrl, strParams, strCookies, ``, header)
	fmt.Println(loginResp)
  //请求passport
	passportResp, _ := callPassport(loginResp, strCookies+";"+loginCookies)
  fmt.Println(passportResp)
	uniqueid := MatchData(passportResp, `"uniqueid":"(.*?)"`)
	homeUrl := "http://weibo.com/u/" + uniqueid + "/home?topnav=1&wvr=6"

	//进入个人主页
	//entryHome(homeUrl, loginCookies)
	//抓取个首页
	fmt.Println(homeUrl)
	return loginCookies
}

func inputcgi(inputDone chan string){
  reader := bufio.NewReader(os.Stdin)
  //for {
	  fmt.Println("waiting for input captcha...")
	  data, _, _ := reader.ReadLine()
		fmt.Println("cmd " + string(data))
	  inputDone <- string(data)
  //}
}

/*
 * crypto passwd for logining
 * var RSAKey = new sinaSSOEncoder.RSAKey();
 * RSAKey.setPublic(me.rsaPubkey, "10001");
 * password = RSAKey.encrypt([me.servertime, me.nonce].join("\t") + "\n" + password)
 *
 */
func encryptPassword(loginInfo map[string]interface{}, password string) string {
  z := new(big.Int)
	z.SetString(loginInfo["pubkey"].(string), 16)
	pub := rsa.PublicKey{
		N: z,
		E: 65537,
	}
	encryString := strconv.Itoa(int(loginInfo["servertime"].(float64))) + "\t" + loginInfo["nonce"].(string) + "\n" + password
	encryResult, _ := rsa.EncryptPKCS1v15(rand.Reader, &pub, []byte(encryString))
	return hex.EncodeToString(encryResult)
}

/*
 * open main page and you should get cookie and save
 */
 func getCookies() string{
   loginUrl := `http://weibo.com/login.php`
   _, strCookies := DoRequest(`GET`, loginUrl, ``, ``, ``, nil)
   return strCookies
 }

/*
 * when finish inputing the username, send the prelogin req
 * you can get login info for logining sina
 */
func getPreLogin(su string) map[string]interface{} {
  preLoginUrl := `https://login.sina.com.cn/sso/prelogin.php?entry=weibo&callback=sinaSSOController.preloginCallBack&su=`+
  su + `&rsakt=mod&checkpin=1&client=ssologin.js(v1.4.18)&_=`
  resBody, _ := DoRequest(`GET`, preLoginUrl, ``, ``, ``, nil)
  //use regex extra json string
  strLoginInfo := RegexFind(resBody, `\((.*?)\)`)
  fmt.Println("======getPreLogin")
  //parse json str to map[string]string
  //json str 转map
	var loginInfo map[string]interface{}
	if err := json.Unmarshal([]byte(strLoginInfo), &loginInfo); err == nil {
    //return nil
	}
  return loginInfo
}

/*
 * entry:weibo
 * gateway:1
 * from:
 * savestate:7
 * useticket:1
 * pagerefer:
 * vsnf:1
 * su:aGZ1dGN4JTQwMTYzLmNvbQ==
 * service:miniblog
 * servertime:1477206529
 * nonce:2D9O10
 * pwencode:rsa2
 * rsakv:1330428213
 * sp:b96481646e643b59373c8b706e439c5f5b95990b7110e62e7f7e67ccab81571fc2e216950c6bf5764e181c2735839eb161d074ea489d2254be4a6756e05745a5fde469f30d3ae23539d1c74d321f08fc169e08f2f5da9f49c9f7e40e17c5a3d278b6bfcca214c70ed4fd37cb75c8d0e4a8d30fe671c418fc5a256305c93bafd0
 * sr:1280*800
 * encoding:UTF-8
 * prelt:839
 * url:http://weibo.com/ajaxlogin.php?framelogin=1&callback=parent.sinaSSOController.feedBackUrlCallBack
 * returntype:META
 */
func buildParems(su, sp, captcha string, loginInfo map[string]interface{}) string {

  strParams := "entry=weibo&gateway=1&from=&savestate=7&useticket=1&pagerefer=&vsnf=1&su=" +
  su + "&service=miniblog&servertime=" + strconv.Itoa(int(loginInfo["servertime"].(float64))) +
  "&nonce=" + loginInfo["nonce"].(string) +
  "&pwencode=rsa2&rsakv=" + loginInfo["rsakv"].(string) +
  "&sp=" + sp +
  "&sr=1280*800&encoding=UTF-8&prelt=839&url=http%3A%2F%2Fweibo.com%2Fajaxlogin.php%3Fframelogin%3D1%26callback%3Dparent.sinaSSOController.feedBackUrlCallBack&returntype=META"
  //需要验证码
	if loginInfo["showpin"].(float64) == 1 {
		strParams += "&pcid=" + loginInfo["pcid"].(string)
		strParams += "&door=" + captcha
	}
	fmt.Println("buildParems " + strParams)
  return strParams
}

//获取passport并请求
func callPassport(resp, cookies string) (passresp, passcookies string) {

	//提取passport跳转地址
	passportUrl := RegexFind(resp, `location.replace\(\'(.*?)\'\)`)
	passresp, passcookies = DoRequest(`GET`, passportUrl, ``, cookies, ``, header)
	return
}

//进入首页
func entryHome(redirectUrl, cookies string) (homeResp, homeCookies string) {
  fmt.Println("======entryHome: " + redirectUrl)
	homeResp, homeCookies = DoRequest(`GET`, redirectUrl, ``, cookies, ``, header)
	return
}

//保存验证码
func saveCaptcha(pcid, cookies string) {
	rnd := strconv.Itoa(int(time.Now().Unix()))
	captchUrl := "http://login.sina.com.cn/cgi/pin.php?r=" + rnd + "&s=0&p=" + pcid
	fmt.Println(captchUrl)
	captcha, _ := DoRequest(`GET`, captchUrl, ``, cookies, ``, nil)
	imgSave, err := os.Create("./" + rnd + ".png")
	if err != nil {
		fmt.Println(err.Error())
	}
	io.Copy(imgSave, bytes.NewReader([]byte(captcha)))
	fmt.Println("saveCaptcha")
}

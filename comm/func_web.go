package comm

import (
	"fmt"
	"github.com/Yq2/lottery/conf"
	"github.com/Yq2/lottery/models"
	"net/http"
	"net/url"
	"strconv"
)

// 得到客户端IP地址
func ClientIP(request *http.Request) string {
	//host, _, _ := net.SplitHostPort(request.RemoteAddr)
	//模拟访问IP
	host := fmt.Sprintf("%d.%d.%d.%d",
		Random(256),
		Random(256),
		Random(256),
		Random(256),)
	return host
}

// 跳转URL
// 301表示永久性转移
// 302表示临时转移
func Redirect(writer http.ResponseWriter, url string) {
	writer.Header().Add("Location", url)
	writer.WriteHeader(http.StatusFound)
}

// 从cookie中得到当前登录的用户
func GetLoginUser(request *http.Request) *models.ObjLoginuser {
	uid := Random(1000000)
	loginuser := &models.ObjLoginuser{
		Uid:uid,
		Username:fmt.Sprintf("http_test_uid_%d",uid),
		Now:NowUnix(),
		Ip:ClientIP(request),
	}
	sign := createLoginuserSign(loginuser)
	loginuser.Sign = sign
	/*
	c, err := request.Cookie("lottery_loginuser")
	if err != nil {
		return nil
	}
	//将cookie转换 Values这种map结构
	//c.Value是string类型的: a=1&b=3
	params, err := url.ParseQuery(c.Value)
	if err != nil {
		return nil
	}
	uid, err := strconv.Atoi(params.Get("uid"))
	if err != nil || uid < 1 {
		return nil
	}
	// Cookie 最长使用时长30天
	now, err := strconv.Atoi(params.Get("now"))
	//如果cookie里面时间超过30天
	if err != nil || NowUnix()-now > 86400*30 {
		return nil
	}
	//// IP修改了是不是要重新登录
	//ip := params.Get("ip")
	//if ip != ClientIP(request) {
	//	return nil
	//}
	// 登录信息
	loginuser := &models.ObjLoginuser{}
	loginuser.Uid = uid
	loginuser.Username = params.Get("username")
	loginuser.Now = now
	loginuser.Ip = ClientIP(request)
	loginuser.Sign = params.Get("sign")
	if err != nil {
		log.Println("fuc_web GetLoginUser Unmarshal ", err)
		return nil
	}
	//sign表示签名字符串
	sign := createLoginuserSign(loginuser)
	//验证用户信息签名
	if sign != loginuser.Sign {
		log.Println("fuc_web GetLoginUser createLoginuserSign not sign", sign, loginuser.Sign)
		return nil
	}
	*/
	return loginuser
}

// 将登录的用户信息设置到cookie中
func SetLoginuser(writer http.ResponseWriter, loginuser *models.ObjLoginuser) {
	if loginuser == nil || loginuser.Uid < 1 {
		c := &http.Cookie {
			Name:   "lottery_loginuser",
			Value:  "",
			Path:   "/",
			MaxAge: -1, //MaxAge为负数表示浏览器关闭会话失效，0表示立即删除cookie
		}
		//往响应体里面写入cookie
		http.SetCookie(writer, c)
		return
	}
	if loginuser.Sign == "" {
		loginuser.Sign = createLoginuserSign(loginuser)
	}
	params := url.Values{}
	params.Add("uid", strconv.Itoa(loginuser.Uid))
	params.Add("username", loginuser.Username)
	params.Add("now", strconv.Itoa(loginuser.Now))
	params.Add("ip", loginuser.Ip)
	params.Add("sign", loginuser.Sign)
	c := &http.Cookie {
		Name:  "lottery_loginuser",
		Value: params.Encode(), //将map编码成字符串
		Path:  "/", //根目录下有小
	}
	//往响应体里面写入cookie，返还给客户端
	http.SetCookie(writer, c)
}

// 根据登录用户信息生成加密字符串
func createLoginuserSign(loginuser *models.ObjLoginuser) string {
	str := fmt.Sprintf("uid=%d&username=%s&secret=%s", loginuser.Uid, loginuser.Username, conf.CookieSecret)
	//对字符串进行MD5签名
	return CreateSign(str)
}
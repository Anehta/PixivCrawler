package main

import (
	"strconv"
	//	"io/ioutil"
	"fmt"
	"log"

	"github.com/Unknwon/goconfig"
)

func main() {
	conf, err := goconfig.LoadConfigFile("user.ini")
	if err != nil {
		log.Println(err)
		fmt.Println("估计是没找到user.ini这个文件")
		return
	}
	//	return
	username, uerr := conf.GetValue(goconfig.DEFAULT_SECTION, "username")
	if uerr != nil {
		log.Println("用户名错误")
		return
	}
	password, perr := conf.GetValue(goconfig.DEFAULT_SECTION, "password")
	if perr != nil {
		log.Println("密码错误")
		return
	}

	threadstr, terr := conf.GetValue(goconfig.DEFAULT_SECTION, "thread")
	if terr != nil {
		log.Println("线程设置错误,默认30线程")
	}
	thread, tserr := strconv.Atoi(threadstr)
	if tserr != nil {
		log.Println("thread转换失败")
		thread = 30
	}

	fmt.Println("用户:", username)
	//	fmt.Println("密码:", password)
	//	return
	fmt.Println("垃圾p站,网速真慢,要是出现莫名其妙的错误的话重新开就行")
	fmt.Println("这个程序主要用来爬用户的关注列表的人的图片，默认开启30个线程下载,实际可以在user.ini中的thread属性里调整,下载速度视网速情况，最好别中途断开。")
	fmt.Println("作者Anehta qq:479673181")
	fmt.Println("注意把用户名和密码填对，懒得做错误检测了")
	test := &Pixiv{}
	test.Thread = thread
	test.LoginPixiv(username, password)
	userlist, err := test.GetFollow()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(userlist)
	lis, _ := test.GetPhotoFromUserList(userlist)
	//	fmt.Println(lis)
	//	var photoList []Photo
	//	var photo Photo
	//	photo.Author = "秋鮭"
	//	photo.Url = "http://www.pixiv.net/member_illust.php?mode=medium&illust_id=55246717"
	//	photo.Name = "渾身の唇_"
	//	photoList = append(photoList, lis)

	test.DownloadFromPhotoList(userlist, lis)
}

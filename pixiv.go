package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"os"
	"regexp"
	"sync"
	"time"
	//	"log"
	"math"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"

	"github.com/opesun/goquery"
)

type User struct {
	Name string
	Id   string
}
type Pixiv struct {
	client *http.Client
	Thread int
}

type Photo struct {
	Url    string
	Author string
	Name   string
}

func (self *Pixiv) LoginPixiv(username string, password string) error {
	res, _ := http.NewRequest("POST", "https://www.pixiv.net/login.php", bytes.NewBuffer([]byte("mode=login&return_to=/&pixiv_id="+username+"&pass="+password+"&skip=1")))
	res.Header.Set("Accept-Language", "zh-Hans-CN,zh-Hans;q=0.5")
	res.Header.Set("Content-Length", "41")
	res.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, perr := url.Parse("http://280761.imwork.net:41031")

	if perr != nil {
		fmt.Println("代理服务器解析失败")
	}
	self.client = &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				deadline := time.Now().Add(100 * time.Second)
				c, err := net.DialTimeout(netw, addr, time.Second*100)

				if err != nil {
					return nil, err
				}
				c.SetDeadline(deadline)
				return c, nil
			},
			//			Proxy: http.ProxyURL(proxyurl),
		}}
	self.client.Jar, _ = cookiejar.New(nil)
	_, err := self.client.Do(res)
	if err != nil {
		return errors.New("Error:登陆失败")
	}
	res.Body.Close()
	return nil
}

func (self *Pixiv) GetQueryFromUrl(url string) (*goquery.Nodes, error) {
	resp, err := self.client.Get(url)
	if err != nil {
		//		return self.GetQueryFromUrl(url)
		//		log.Println(err)
		//		log.Println("傻逼p站网速真慢")
		return self.GetQueryFromUrl(url)
	}

	result, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return self.GetQueryFromUrl(url)
	}

	node, err := goquery.ParseString(string(result))
	if err != nil {
		return self.GetQueryFromUrl(url)
	}
	return &node, nil
}

//
func (self *Pixiv) GetFollow() ([]User, error) {
	resp, err := self.client.Get("http://www.pixiv.net/bookmark.php?type=user")
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Error:获取正在关注列表失败")
	}

	result, err := ioutil.ReadAll(resp.Body)
	node, err := goquery.ParseString(string(result))
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Error:解析html失败->获取关注列表")
	}
	find := node.Find(".info")
	count_text := find.Find(".count").Text()
	count_text = strings.Replace(count_text, " ", "", -1)
	count, err := strconv.Atoi(count_text)
	if err != nil {
		fmt.Println(err)
		fmt.Println(string(result))
		return nil, errors.New("Error:解析html失败->转换count失败")
	}

	fmt.Println("关注人数为:", count)

	row := 16.0
	col := 3.0

	page := int(math.Ceil(float64(count) / (row * col)))

	var waitgroup sync.WaitGroup
	user_chan := make(chan *[]*User, page)

	for i := 0; i < page; i++ {
		waitgroup.Add(1)
		go func(index int) {
			defer waitgroup.Done()
			fmt.Println("正在获取第", index+1, "页")
			user_result_text, err := self.client.Get("http://www.pixiv.net/bookmark.php?type=user&rest=show&p=" + strconv.Itoa(index+1))
			if err != nil {
				fmt.Println("Error:获取page:", index+1, "失败:", err)
				return
			}
			user_result, err := ioutil.ReadAll(user_result_text.Body)
			if err != nil {
				fmt.Println("Error:转换page:", index+1, "失败:", err)
				return
			}
			user_node, err := goquery.ParseString(string(user_result))
			if err != nil {
				fmt.Println("Error:解析html失败->page:", index+1)
				return
			}

			member_node := user_node.Find(".members")
			ul_node := member_node.Find("ul")
			//			fmt.Println("ul_node", ul_node.Text())
			li_node := ul_node.Find("li")
			li_count := li_node.Length()
			fmt.Println("第", index+1, "页有", li_count, "个人")
			var userlist []*User
			for j := 0; j < li_count; j++ {
				data := li_node.Eq(j)
				userdata_node := data.Find(".userdata")
				a := userdata_node.Find(".ui-profile-popup")
				href_url := a.Attr("data-user_id")
				username := a.Attr("data-user_name")
				user := &User{}
				user.Id = href_url
				//				fmt.Println(username)
				user.Name = username
				userlist = append(userlist, user)

			}
			user_chan <- &userlist
		}(i)
	}

	waitgroup.Wait()
	var main_userlist []User
	for {
		select {
		case ul := <-user_chan:
			for _, v := range *ul {
				var user User
				user.Id = v.Id
				user.Name = v.Name
				main_userlist = append(main_userlist, user)
			}
			break

		case <-time.After(time.Millisecond * 10):
			return main_userlist, nil
			break
		}
	}

	if len(main_userlist) != count {
		log.Println("网络异常，获取关注列表人数不匹配，重新获取")
		return self.GetFollow()
	}
	return main_userlist, nil
}

func CheckName(name string) string {
	name = strings.Replace(name, "\\", "0", -1)
	name = strings.Replace(name, "/", "1", -1)
	name = strings.Replace(name, ":", "2", -1)
	name = strings.Replace(name, "?", "3", -1)
	name = strings.Replace(name, "*", "4", -1)
	name = strings.Replace(name, "\"", "5", -1)
	name = strings.Replace(name, "|", "6", -1)
	name = strings.Replace(name, "<", "7", -1)
	name = strings.Replace(name, ">", "8", -1)
	return name
}

func (self *Pixiv) GetPhotoFromUserList(userList []User) ([]Photo, error) {
	var repeatname map[string]int
	repeatname = make(map[string]int)
	wait := &sync.WaitGroup{}
	var tmpphotos []Photo
	p := &sync.RWMutex{}

	for _, v := range userList {
		wait.Add(1)
		go func(user User) {
		Tag:
			Url := "http://www.pixiv.net/member_illust.php?id=" + user.Id + "&type=all&p=1"
			row := 4.0
			col := 5.0
			nodes, err := self.GetQueryFromUrl(Url)

			if err != nil {
				//				log.Println(err)
				goto Tag
			}
			count_str := nodes.Find(".count-badge").Text()
			if count_str == "" {
				err := errors.New("Error:获取用户->" + user.Name + "的作品个数失败,正在尝试重新获取")
				log.Println(err)
				goto Tag
			}

			count_str = strings.Replace(count_str, "件", "", -1)
			count, err := strconv.Atoi(count_str)
			if count_str == "" {
				err := errors.New("Error:转换数字失败")
				log.Panicln(err)
			}

			page := int(math.Ceil(float64(count) / (row * col)))

			image_items := nodes.Find(".image-item")

			for k := 0; k < image_items.Length(); k++ {
				image_item_node := image_items.Eq(k)
				work_node := image_item_node.Find(".work")
				photo_url := "http://www.pixiv.net" + work_node.Attr("href")
				title_node := image_item_node.Find(".title")
				name := title_node.Text()

				for {
					p.RLock()
					if _, ok := repeatname[name]; ok {
						p.RUnlock()
						name = name + "_"
					} else {
						p.RUnlock()
						break
					}
				}
				var tmp Photo
				name = CheckName(name)
				tmp.Name = name
				p.Lock()
				repeatname[name] = 1
				p.Unlock()

				tmp.Url = photo_url
				tmp.Author = user.Name
				tmpphotos = append(tmpphotos, tmp)
				//				fmt.Println(tmp)
			}

			photo_wait := &sync.WaitGroup{}

			for j := 1; j < page; j++ {
				photo_wait.Add(1)
				go func(pageindex int, authorname string, userid string) {
				STAG:
					PageUrl := "http://www.pixiv.net/member_illust.php?id=" + userid + "&type=all&p=" + strconv.Itoa(pageindex)
					jnodes, errs := self.GetQueryFromUrl(PageUrl)
					if errs != nil {
						//						log.Println(PageUrl)
						goto STAG
					}
					jimage_items := jnodes.Find(".image-item")
					for k := 0; k < image_items.Length(); k++ {
						image_item_node := jimage_items.Eq(k)
						work_node := image_item_node.Find(".work")
						photo_url := "http://www.pixiv.net" + work_node.Attr("href")
						title_node := image_item_node.Find(".title")
						name := title_node.Text()
						for {
							p.RLock()
							if _, ok := repeatname[name]; ok {
								p.RUnlock()
								name = name + "_"
							} else {
								p.RUnlock()
								break
							}
						}
						var tmp Photo
						name = CheckName(name)
						tmp.Name = name

						p.Lock()
						repeatname[name] = 1
						p.Unlock()
						tmp.Url = photo_url
						tmp.Author = authorname
						tmpphotos = append(tmpphotos, tmp)
						//					fmt.Println(tmp)
					}
					photo_wait.Done()
				}(j, user.Name, user.Id)
			}
			photo_wait.Wait()
			fmt.Println("读取"+user.Name+"成功|"+user.Name+"有", count, "件作品")
			wait.Done()
		}(v)
	}

	wait.Wait()

	return tmpphotos, nil
}

func (self *Pixiv) DownloadFromPhotoList(userList []User, photoList []Photo) {
	fmt.Println("开始下载图片")
	os.Mkdir("关注列表", 0666)
	for _, v := range userList {
		os.Mkdir("关注列表/"+v.Name, 0666)
	}

	wait := &sync.WaitGroup{}
	fuckpixiv := &sync.Mutex{}

	download_count := 0
	download_sum := len(photoList)
	co := 0
	for _, v := range photoList {
		if co < self.Thread {
			co++
			wait.Add(1)

			//		wait.Add(1)
			go func(info Photo) {
				defer wait.Done()
				//		go func(info Photo) {
				//		fmt.Println("cao")
			PTAG:
				nodes, err := self.GetQueryFromUrl(info.Url)
				if err != nil {
					//				log.Println("PTAG", err)
					goto PTAG
				}
				original_image := nodes.Find(".original-image")
				image_url := original_image.Attr("data-src")
				image_url = strings.Replace(image_url, " ", "", -1)
				if image_url == "" {
					return
				}
				//			fmt.Println(image_url)
			SDTAG:
				res, err := http.NewRequest("GET", image_url, bytes.NewBuffer([]byte("")))

				if err != nil {
					fmt.Println(err)
					goto SDTAG
				}
				res.Header.Set("Accept", "image/png, image/svg+xml, image/jxr, image/*;q=0.8, */*;q=0.5")
				res.Header.Set("Referer", info.Url)
				//		res.Header.Set("Accept-Language", "Accept-Language: zh-Hans-CN,zh-Hans;q=0.5")
				//		res.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2486.0 Safari/537.36 Edge/13.10586")
				//			res.Header.Set("Accept-Encoding", "gzip, deflate")
				//		res.Header.Add("Host", "i2.pixiv.net")
				resp, err := self.client.Do(res)

				if err != nil {
					//				log.Println("SDTAG.Url:", info.Url)
					goto SDTAG
				}
				if resp == nil {
					log.Println("图片获取不完整resp == nil", info.Url)
					goto SDTAG
				}
				if resp.StatusCode != http.StatusOK {
					log.Println("图片获取不完整StatusCode!=http.StatusOk,Url:", info.Url)
					goto SDTAG
				}

				//		fmt.Println("cao")

				data, derr := ioutil.ReadAll(resp.Body)
				//			fmt.Println("原大小", resp.ContentLength, "实际大小", len(data))
				if int64(len(data)) != resp.ContentLength {
					//				fmt.Println("图片获取缺失,正在重新获取,原大小:", resp.ContentLength, "实际大小:", len(data))
					goto SDTAG
				}
				if derr != nil {
					log.Println("读取图片失败,正在重新读取,error:", derr, "URL:"+image_url)
					resp.Body.Close()
					goto SDTAG
				}
				//		fmt.Println(string(data))
				reg := regexp.MustCompile(".jpg|.png|.gif")
				strs := reg.FindAllString(image_url, -1)
				if len(strs[0]) < 1 {
					log.Println("正则表达式匹配失败:", image_url)
				}
				hz := strs[0]
				f, ferr := os.Create("关注列表/" + info.Author + "/" + info.Name + hz)
				if ferr != nil {
					log.Println("创建文件失败,error:", ferr, "URL:"+image_url)
					fuckpixiv.Lock()
					download_count++
					fuckpixiv.Unlock()
					f.Close()
					return
				}
				io.Copy(f, bytes.NewReader(data))

				fuckpixiv.Lock()
				download_count++
				fmt.Println("下载", info.Author, "的大作", info.Name, "完成,进度:", download_count, "/", download_sum)
				fuckpixiv.Unlock()
				resp.Body.Close()
				f.Close()

				//				wait.Done()
			}(v)
		} else {
			co = 0
			wait.Wait()
		}
		//		}(v)
	}

	wait.Wait()
	fmt.Println("全部下载完成")
}
